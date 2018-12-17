//  Copyright (c) 2018 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package snapshot

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/uber/makisu/lib/fileio"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/mountutils"
	"github.com/uber/makisu/lib/pathutils"
	"github.com/uber/makisu/lib/tario"

	"github.com/andres-erbsen/clock"
)

// memFSNode represents one node of the directory tree in the merged fs view.
type memFSNode struct {
	*contentMemFile                       // No whiteouts
	children        map[string]*memFSNode // Child nodes of the directory, indexed by base name
}

// newMemFSNode inits a new memFSNode instance.
func newMemFSNode(mf *contentMemFile) *memFSNode {
	return &memFSNode{mf, make(map[string]*memFSNode)}
}

// isOnDisk returns true if the path exists on disk.
func (n *memFSNode) isOnDisk() (bool, error) {
	if _, err := os.Lstat(n.src); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

// MemFS contains a collection of in-memory layers and a merged fs view.
type MemFS struct {
	clk  clock.Clock
	tree *memFSNode

	blacklist []string
	layers    []*memLayer
}

// NewMemFS inits a new MemFS instance.
func NewMemFS(clk clock.Clock, root string, blacklist []string) (*MemFS, error) {
	fi, err := os.Lstat(root)
	if err != nil {
		return nil, fmt.Errorf("unable to stat root dir: %s", root)
	}
	hdr, err := newMemLayer().createHeader(root, root, "/", fi)
	if err != nil {
		return nil, fmt.Errorf("unable to create root header")
	}
	return &MemFS{
		clk:       clk,
		tree:      newMemFSNode(newContentMemFile(root, "/", hdr)),
		blacklist: blacklist,
	}, nil
}

// Reset resets the in-memory file system view of the memFS.
func (fs *MemFS) Reset() {
	fs.tree.children = make(map[string]*memFSNode)
}

// Checkpoint relocates the given src files & directories to the given newRoot.
func (fs *MemFS) Checkpoint(newRoot string, sources []string) error {
	resolvedSources := []string{}
	for _, src := range sources {
		if matches, err := filepath.Glob(src); err != nil || len(matches) == 0 {
			resolvedSources = append(resolvedSources, src)
		} else {
			resolvedSources = append(resolvedSources, matches...)
		}
	}

	log.Infof("* Moving directories %v to %s", sources, newRoot)
	copier := fileio.NewCopier(fs.blacklist)
	for _, src := range resolvedSources {
		if !filepath.IsAbs(src) {
			src = filepath.Join(fs.tree.src, src)
		}
		trimmedSrc, err := pathutils.TrimRoot(src, fs.tree.src)
		if err != nil {
			return fmt.Errorf("trim src %s: %s", src, err)
		}
		dst := filepath.Join(newRoot, trimmedSrc)
		sourceInfo, err := os.Stat(src)
		if err != nil {
			return fmt.Errorf("stat %s: %s", src, err)
		}
		if sourceInfo.IsDir() {
			if err := copier.CopyDir(src, dst, 0, 0); err != nil {
				return fmt.Errorf("copy dir %s: %s", src, err)
			}
		} else {
			if err := copier.CopyFile(src, dst, 0, 0); err != nil {
				return fmt.Errorf("copy file %s: %s", src, err)
			}
		}
	}

	return nil
}

// Remove removes everything under the root of the memFS.
func (fs *MemFS) Remove() error {
	return removeAllChildren(fs.tree.src, fs.blacklist)
}

// UpdateFromTarPath updates MemFS with the contents of the tarball at the given
// path. untars the tarball onto the root of MemFS 'untar' specifies if the
// contents should also be written to disk at the MemFS root or not.
func (fs *MemFS) UpdateFromTarPath(source string, untar bool) error {
	reader, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open tar file: %s", err)
	}
	defer reader.Close()
	gzipReader, err := tario.NewGzipReader(reader)
	if err != nil {
		return fmt.Errorf("new gzip reader: %s", err)
	}
	return fs.UpdateFromTarReader(tar.NewReader(gzipReader), untar)
}

// UpdateFromTarReader updates MemFS with the contents of the tarball from the
// gvien reader, and optionally untars the tarball onto the root of MemFS.
func (fs *MemFS) UpdateFromTarReader(r *tar.Reader, untar bool) error {
	start := time.Now()
	// Keep a list of all hard links that we will create in a second pass.
	hardlinks := make(map[string]*tar.Header)

	// Also keep a list of the mod times of the parent directories. We will use this to
	// reset them.
	modtimes := make(map[string]time.Time)

	var count int
	l := newMemLayer()
	for {
		hdr, err := r.Next()
		if err == io.EOF {
			duration := time.Since(start).Round(time.Millisecond)
			if untar {
				log.Infof("* Untarred %d files to %s in %v", count, fs.tree.src, duration)
			}
			break
		} else if err != nil {
			return fmt.Errorf("read header: %s", err)
		}

		path := filepath.Join(fs.tree.src, hdr.Name)
		if skip, err := shouldSkip(path, hdr.FileInfo(), fs.blacklist); err != nil {
			return fmt.Errorf("check if should skip %s: %s", path, err)
		} else if skip {
			continue
		} else if isMounted, err := mountutils.IsMounted(path); err != nil {
			return fmt.Errorf("check if mounted %s: %s", path, err)
		} else if isMounted {
			continue
		}

		// Record the modtime of the parent directory to reset it after we deal with all of
		// the other files. If we are not untarring, this is not necessary and may fail
		// because not all files are necessarily on disk.
		if untar {
			parentDir := filepath.Dir(path)
			if _, found := modtimes[parentDir]; !found {
				parentFi, err := os.Lstat(parentDir)
				if err != nil {
					return fmt.Errorf("stat parent dir of %s: %s", path, err)
				}
				modtimes[parentDir] = parentFi.ModTime()
			}
		}

		hdr.Name = pathutils.RelPath(hdr.Name)

		// If the new file is a hard link, then append it to the list
		// that will be created later.
		if hdr.Typeflag == tar.TypeLink {
			// Docker hard link names are all absolute, but don't have a leading slash.
			hdr.Linkname = pathutils.AbsPath(hdr.Linkname)
			hardlinks[path] = hdr
		} else {
			if untar {
				if err := fs.untarOneItem(path, hdr, r); err != nil {
					return fmt.Errorf("untar one item %s: %s", path, err)
				}
			}
			if err := fs.maybeAddToLayer(l, "", pathutils.AbsPath(hdr.Name), hdr, false); err != nil {
				return fmt.Errorf("add hdr from tar to layer: %s", err)
			}
		}
		count++
	}

	// Run through all the hard links and create them.
	for path, hdr := range hardlinks {
		if untar {
			if err := fs.untarOneItem(path, hdr, nil); err != nil {
				return fmt.Errorf("untar one item %s: %s", path, err)
			}
		}
		if err := fs.maybeAddToLayer(l, "", pathutils.AbsPath(hdr.Name), hdr, false); err != nil {
			return fmt.Errorf("add hdr from tar to layer: %s", err)
		}
	}

	// Reset the mod times on all of the directory we changed.
	for path, modtime := range modtimes {
		if err := os.Chtimes(path, modtime, modtime); err != nil {
			return fmt.Errorf("chtimes on parent directory %s: %s", path, err)
		}
	}
	fs.layers = append(fs.layers, l)
	log.Infof("* Merged %d headers from tar to memfs", l.count())
	return nil
}

// AddLayerByScan creates an in-memory layer by scanning the differences
// between the file system and existing in-memory merged layers. The
// resulting layer is merged in memory and written to the tar writer.
func (fs *MemFS) AddLayerByScan(w *tar.Writer) error {
	fs.sync()
	if l, err := fs.createLayerByScan(); err != nil {
		return fmt.Errorf("create layer by scan: %s", err)
	} else if err := fs.commitLayer(l, w); err != nil {
		return fmt.Errorf("commit layer by scan: %s", err)
	} else {
		log.Infof("* Created layer by scanning filesystem; %d files found", l.count())
	}
	return nil
}

// AddLayerByCopyOps creates an in-memory layer by performing copy operations
// on the given src-dst pairs. The file system is not modified during this
// operation. The resulting layer is merged in memory and written to the
// tar writer.
func (fs *MemFS) AddLayerByCopyOps(cs []*CopyOperation, w *tar.Writer) error {
	fs.sync()
	l := newMemLayer()
	for _, c := range cs {
		if err := fs.addToLayer(l, c); err != nil {
			return fmt.Errorf("create layer by copy ops: %s", err)
		}
	}
	if err := fs.commitLayer(l, w); err != nil {
		return fmt.Errorf("commit layer by copy ops: %s", err)
	}
	log.Infof("* Created copy layer with %d files", l.count())
	return nil
}

// sync flushes filesystem cache, so mtime would be guaranteed to be updated.
// It also waits at least one sec, in case mtime doesn't have sub-second
// resolution.
func (fs *MemFS) sync() {
	// Ensure this function takes at least one sec.
	block := make(chan interface{}, 1)
	go func() {
		<-time.After(1 * time.Second)
		close(block)
	}()

	// Flush cache.
	syscall.Sync()

	<-block
}

// createLayerByScan computes the differences between the file system and merged
// layers in memory, updating MemFS as it goes and returning the diffs as a single layer.
func (fs *MemFS) createLayerByScan() (*memLayer, error) {
	start := time.Now()
	log.Info("* Collecting filesystem diff")

	l := newMemLayer()
	root := fs.tree.src
	if err := walk(
		root, fs.blacklist, func(src string, fi os.FileInfo) error {
			dst, err := pathutils.TrimRoot(src, root)
			if err != nil {
				return err
			}
			hdr, err := l.createHeader(fs.tree.src, src, dst, fi)
			if err != nil {
				return fmt.Errorf("create header %s: %s", dst, err)
			}
			if err := fs.maybeAddToLayer(l, src, dst, hdr, true); err != nil {
				return fmt.Errorf("add to layer: %s", err)
			}
			return nil
		}); err != nil {
		return nil, fmt.Errorf("walk %s: %s", root, err)
	}

	duration := time.Since(start).Round(time.Millisecond)
	log.Infof("* Finished collecting diff in %s: %d files found", duration, l.count())
	return l, nil
}

// addToLayer computes the in-memory differences created by the copy operation,
// updating MemFS as it goes and returning the diffs as a single layer.
// There are 3 cases:
// 1) /source/file1 /target/file2
//   - file1 copied to file2
// 2) /source/dir1  /target/dir2
//   - contents of dir1 copied to dir2
// 3) /source1/file1, /source2/dir1, ...   /target/dir2/
//   - files copied to dir2
//   - contents of dirs copied to dir2
func (fs *MemFS) addToLayer(l *memLayer, c *CopyOperation) error {
	var err error
	createDst := true

	if len(c.srcs) == 1 {
		src := filepath.Join(c.srcRoot, c.srcs[0])
		if fi, err := os.Stat(src); err != nil {
			return fmt.Errorf("stat src %s: %s", src, err)
		} else if !fi.IsDir() {
			// Case 1, no need to ensure dst exists explicitly.
			createDst = false
		}
	}
	if createDst {
		// Ensure dst either already exists or create it with default
		// permissions, and update dst by following symlinks.
		resolved, err := fs.addAncestors(l, pathutils.AbsPath(c.dst), true, 0, c.uid, c.gid)
		if err != nil {
			return fmt.Errorf("add ancestors of %s: %s", c.dst, err)
		}
		if !strings.HasSuffix(resolved, "/") {
			resolved += "/"
		}
		c.dst = resolved
	}

	for _, src := range c.srcs {
		src, err = evalSymlinks(src, c.srcRoot)
		if err != nil {
			return fmt.Errorf("eval symlinks for %s: %s", src, err)
		}
		src = filepath.Join(c.srcRoot, src)
		if err := walk(src, nil, func(currSrc string, fi os.FileInfo) error {
			var currDst string
			if currSrc == src {
				if fi.IsDir() {
					// If src is a directory, recursively copy its contents to
					// dst (but not the directory itself since dst directory
					// either already exists or was created at the beginning
					// with default permissions), so continue the walk.
					return nil

				} else if !strings.HasSuffix(c.dst, "/") {
					// If src & dst are files, just copy src to dst (case 1).
					currDst = c.dst

				} else {
					// If src is a file & dst is a dir, copy src to dst/<file>.
					currDst = filepath.Join(c.dst, filepath.Base(src))
				}
			} else {
				// For any path that isn't src itself, copy to the same relative
				// destination in dst (strip src prefix & append to dst).
				currDst = filepath.Join(c.dst, currSrc[len(src):])
			}
			hdr, err := l.createHeader(fs.tree.src, currSrc, currDst, fi)
			if err != nil {
				return fmt.Errorf("create header %s: %s", currDst, err)
			}
			return fs.maybeAddToLayer(l, currSrc, currDst, hdr, false)
		}); err != nil {
			return fmt.Errorf("copy src %s to dst %s: %s", src, c.dst, err)
		}
	}
	return nil
}

// commitLayer writes the layer content into the given tar writer.
// It ensures all paths are alphabetically sorted.
func (fs *MemFS) commitLayer(l *memLayer, w *tar.Writer) error {
	// Write to tar header in alphabetical order.
	if err := l.rangeFiles(func(f memFile) error {
		return f.commit(w)
	}); err != nil {
		return fmt.Errorf("commit layer: %s", err)
	}
	fs.layers = append(fs.layers, l)
	return nil
}

// maybeAddToLayer converts given file into to tar header, and adds to the layer
// if it's different from what's already in the in-memory fs.
// It ensures that all intermediate directories exist.
// Set createWhiteout to false to avoid whiting out files, but that won't
// prevent files/directories from being overwritten.
func (fs *MemFS) maybeAddToLayer(
	l *memLayer, src, dst string, hdr *tar.Header, createWhiteout bool) error {
	// Check if the header already exists and is up-to-date.
	updated, n, err := fs.isUpdated(dst, hdr)
	if err != nil {
		return fmt.Errorf("check header %s: %s", dst, err)
	} else if updated {
		if dst != "/" { // Root itself is not added to layers.
			// Add intermediate directories for changed file.
			if _, err := fs.addAncestors(l, pathutils.AbsPath(dst), false, 0, 0, 0); err != nil {
				return fmt.Errorf("add ancestors of %s: %s", dst, err)
			}
			// Add changed file.
			if err := l.addHeader(src, dst, hdr).updateMemFS(fs.tree); err != nil {
				return fmt.Errorf("update memfs with file %s: %s", dst, err)
			}
		}
	}

	if createWhiteout {
		// Handle deletions.
		// Note: Only one whiteout file is needed for a deleted subtree.
		if hdr.Typeflag == tar.TypeDir && n != nil {
			for _, child := range n.children {
				if ok, err := child.isOnDisk(); err != nil {
					return fmt.Errorf("check on disk %s: %s", child.dst, err)
				} else if !ok {
					if mf, err := l.addWhiteout(child.dst); err != nil {
						return fmt.Errorf(
							"add whiteout to layer %s: %s", child.dst, err)
					} else if err := mf.updateMemFS(fs.tree); err != nil {
						return fmt.Errorf(
							"update memfs with whiteout %s: %s", child.dst, err)
						// Add intermediate directories for whited out file.
					} else if _, err := fs.addAncestors(l, child.dst, false, 0, 0, 0); err != nil {
						return fmt.Errorf("add ancestors of %s: %s", child.dst, err)
					}
				}
			}
		}
	}
	return nil
}

// isUpdated checks if the given path is new or updated compared to what's saved
// in memory. it will also return node if the path exists in memory.
// Note: it doesn't follow symlinks.
func (fs *MemFS) isUpdated(p string, hdr *tar.Header) (bool, *memFSNode, error) {
	curr := fs.tree
	parts := pathutils.SplitPath(p)
	for _, part := range parts {
		if n, ok := curr.children[part]; ok {
			curr = n
		} else {
			return true, nil, nil
		}
	}

	similar, err := tario.IsSimilarHeader(curr.hdr, hdr)
	if err != nil {
		return false, nil, fmt.Errorf("compare header %s: %s", p, err)
	}
	return !similar, curr, nil
}

// addAncestors adds a memFile to the layer for each ancestor of the given path.
// Set inclusive to true to include the dst path itself as a directory.
// It follows symlinks, and returns the resolved dst path to the best of its
// knowledge.
func (fs *MemFS) addAncestors(l *memLayer, dst string, inclusive bool, depth, uid, gid int) (string, error) {
	if depth >= 1024 {
		return "", fmt.Errorf("symlink loop at %s", dst)
	}

	lastAncestor := fs.tree

	var i int
	var part string
	curr := fs.tree
	parts := pathutils.SplitPath(dst)
	end := len(parts) - 1
	if inclusive {
		end = len(parts)
	}
	for ; i < end; i++ {
		part = parts[i]
		if n, ok := curr.children[part]; ok {
			if err := l.addHeader(n.src, n.dst, n.hdr).updateMemFS(fs.tree); err != nil {
				return "", fmt.Errorf("update memfs with ancestor %s: %s", n.dst, err)
			}

			switch n.hdr.Typeflag {
			case tar.TypeDir:
				lastAncestor = n
				curr = n
			case tar.TypeSymlink:
				// Add ancestors of symlink target too.
				remaining := filepath.Join(parts[i+1:]...)
				target := filepath.Join(n.hdr.Linkname, remaining)
				resolved, err := fs.addAncestors(l, target, inclusive, depth+1, uid, gid)
				if err != nil {
					return "", fmt.Errorf(
						"get symlink target ancestors %s: %s", target, err)
				}
				return resolved, nil
			}
		} else {
			break
		}
	}

	// Create missing intermediate dir for unresolved part, using param uid/gid.
	for j := i; j < end; j++ {
		curr := pathutils.AbsPath(filepath.Join(parts[:j+1]...))

		// TODO: lastAncestor is not relevant here.
		hdr, err := l.createHeader(fs.tree.src, "", curr, lastAncestor.hdr.FileInfo())
		if err != nil {
			return "", fmt.Errorf("create header %s: %s", curr, err)
		}
		hdr.ModTime = fs.clk.Now()
		hdr.Uid = uid
		hdr.Gid = gid
		if err := l.addHeader("", curr, hdr).updateMemFS(fs.tree); err != nil {
			return "", fmt.Errorf("update memfs with ancestor %s: %s", curr, err)
		}
	}

	return dst, nil
}

// untarOneItem handles untarring a single header from a tar archive to local
// disk. It handles existing files on disk, applying metainfo from the header,
// and writing content.
func (fs *MemFS) untarOneItem(path string, header *tar.Header, r *tar.Reader) error {
	// If it's a whiteout file, there's no need to check existing path on disk.
	if strings.HasPrefix(filepath.Base(path), _whiteoutPrefix) {
		if err := fs.untarWhiteout(path); err != nil {
			return fmt.Errorf("untar dir: %s", err)
		}
		return nil
	}

	headerInfo := header.FileInfo()
	localInfo, err := os.Lstat(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("lstat %s: %s", path, err)
	} else if err == nil {
		var linkTarget string
		if localInfo.Mode()&os.ModeSymlink != 0 {
			linkTarget, err = os.Readlink(path)
			if err != nil {
				return fmt.Errorf("read link %s: %s", linkTarget, err)
			}

			if filepath.IsAbs(linkTarget) {
				linkTarget, err = pathutils.TrimRoot(linkTarget, fs.tree.src)
				if err != nil {
					return fmt.Errorf("trim link %s: %s", linkTarget, err)
				}
			}
		}
		localHeader, err := tar.FileInfoHeader(localInfo, linkTarget)
		if err != nil {
			return fmt.Errorf("create header %s: %s", path, err)
		}

		// If the file is already on disk, nothing needs to be done.
		if similar, err := tario.IsSimilarHeader(localHeader, header); err != nil {
			return fmt.Errorf("compare headers %s: %s", path, err)
		} else if similar {
			return nil
		}

		// For existing directories, only update information instead of deleting.
		// Otherwise we could be deleting /etc, while some underlying mounted files
		// like /etc/resolv.conf cannot be removed.
		if headerInfo.IsDir() && localInfo.IsDir() {
			if err := tario.ApplyHeader(path, header); err != nil {
				return fmt.Errorf("update fi %s: %s", path, err)
			}
			return nil
		}

		// If a different file already exists on the system, remove it so it can be
		// recreated later.
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("clear existing file %s: %s", path, err)
		}
	}

	switch header.Typeflag {
	case tar.TypeDir:
		if err := fs.untarDirectory(path, header); err != nil {
			return fmt.Errorf("untar dir: %s", err)
		}
	case tar.TypeSymlink:
		if err := fs.untarSymlink(path, header); err != nil {
			return fmt.Errorf("untar symlink: %s", err)
		}
	case tar.TypeLink:
		if err := fs.untarHardlink(path, header); err != nil {
			return fmt.Errorf("untar hard link: %s", err)
		}
	default:
		if err := fs.untarFile(path, header, r); err != nil {
			return fmt.Errorf("untar file: %s", err)
		}
	}
	return nil
}

// untarWhiteout removes the contents under the path specified by the whiteout file.
func (fs *MemFS) untarWhiteout(path string) error {
	oldBase := strings.TrimPrefix(filepath.Base(path), _whiteoutPrefix)
	err := os.RemoveAll(filepath.Join(filepath.Dir(path), oldBase))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("untar whiteout: %s", err)
	}
	return nil
}

// untarDirectory creates the directory specified by path and applies the header metadata.
func (fs *MemFS) untarDirectory(path string, header *tar.Header) error {
	if err := os.Mkdir(path, header.FileInfo().Mode()); err != nil {
		return fmt.Errorf("create dir %s: %s", path, err)
	}
	if err := tario.ApplyHeader(path, header); err != nil {
		return fmt.Errorf("update fi %s: %s", path, err)
	}
	return nil
}

// untarSymlink creates the symlink specified by header at path.
func (fs *MemFS) untarSymlink(path string, header *tar.Header) error {
	target := header.Linkname
	if filepath.IsAbs(header.Linkname) {
		target = filepath.Join(fs.tree.src, target)
	}
	if err := os.Symlink(target, path); err != nil {
		return fmt.Errorf("create symlink %s => %s: %s", path, target, err)
	}
	return nil
}

// untarHardlink creates the hard link specified by header at path.
func (fs *MemFS) untarHardlink(path string, header *tar.Header) error {
	target := filepath.Join(fs.tree.src, header.Linkname)
	if err := os.Link(target, path); err != nil {
		return fmt.Errorf(
			"create link %s => %s: %s", path, target, err)
	}
	if err := tario.ApplyHeader(path, header); err != nil {
		return fmt.Errorf(
			"update hard link %s: %s", path, err)
	}
	return nil
}

// untarFile creates the file specified by header at path, copies its content from
// the tar reader, and applies the metadata.
func (fs *MemFS) untarFile(path string, header *tar.Header, r *tar.Reader) error {
	fi := header.FileInfo()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fi.Mode())
	if err != nil {
		return fmt.Errorf("open file %s: %s", path, err)
	}
	defer file.Close()
	if _, err := io.Copy(file, r); err != nil {
		return fmt.Errorf("read from file %s: %s", path, err)
	}
	if err := tario.ApplyHeader(path, header); err != nil {
		return fmt.Errorf("update fi %s: %s", path, err)
	}
	return nil
}
