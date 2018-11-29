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
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/uber/makisu/lib/pathutils"
	"github.com/uber/makisu/lib/tario"
)

// memFile represents one file in an in-memory layer.
type memFile interface {
	updateMemFS(tree *memFSNode) error
	commit(w *tar.Writer) error
}

// contentMemFile represents a MemFile implementation that references on-disk contents.
type contentMemFile struct {
	src string // Location to read content from while creating tar
	dst string // Location to write content to. Key to layer.files
	hdr *tar.Header
}

// newContentMemFile inits a new contentMemFile.
func newContentMemFile(src, dst string, hdr *tar.Header) *contentMemFile {
	return &contentMemFile{
		src: src,
		dst: dst,
		hdr: hdr,
	}
}

// updateMemFS adds a memFSNode containing contentMemFile to the tree rooted at the given node.
// TODO: this function doesn't update root itself.
func (f *contentMemFile) updateMemFS(node *memFSNode) error {
	parts := pathutils.SplitPath(f.dst)
	for i, part := range parts {
		if n, ok := node.children[part]; ok {
			if i == len(parts)-1 {
				node.children[part] = newMemFSNode(f)

				if f.hdr.Typeflag == tar.TypeDir {
					// Copy the children of existing node.
					for k, child := range n.children {
						node.children[part].children[k] = child
					}
				}
			} else {
				node = n
			}
		} else {
			if i == len(parts)-1 {
				node.children[part] = newMemFSNode(f)
			} else {
				return fmt.Errorf("missing intermediate directory %s in %s", part, f.dst)
			}
		}
	}
	return nil
}

// commit writes the contentMemFile's contents to the tar writer.
func (f *contentMemFile) commit(w *tar.Writer) error {
	if err := tario.WriteEntry(w, f.src, f.hdr); err != nil {
		return fmt.Errorf("content commit %s: %s", f.hdr.Name, err)
	}
	return nil
}

// whiteoutMemFile represents a MemFile implementation that deletes contents.
type whiteoutMemFile struct {
	del string // Location to delete. Key to layer.files key
	hdr *tar.Header
}

// newWhiteoutMemFile inits a new whiteoutMemFile.
func newWhiteoutMemFile(deletedPath, whiteoutPath string) *whiteoutMemFile {
	return &whiteoutMemFile{
		del: deletedPath,
		hdr: &tar.Header{Name: pathutils.RelPath(whiteoutPath)},
	}
}

// updateMemFS deletes the memFSNode designated by whiteoutMemFile from the tree rooted at node.
func (f *whiteoutMemFile) updateMemFS(node *memFSNode) error {
	parts := pathutils.SplitPath(f.del)
	for i, part := range parts {
		if n, ok := node.children[part]; ok {
			if i == len(parts)-1 {
				delete(node.children, part)
			} else {
				node = n
			}
		} else {
			if i != len(parts)-1 {
				return fmt.Errorf("missing intermediate dir %s in %s", part, f.del)
			}
			return fmt.Errorf("whiteout nonexistent path %s", f.del)
		}
	}
	return nil
}

// commit writes an empty whiteout file to the tar writer.
func (f *whiteoutMemFile) commit(w *tar.Writer) error {
	if err := tario.WriteHeader(w, f.hdr); err != nil {
		return fmt.Errorf("whiteout commit %s: %s", f.hdr.Name, err)
	}
	return nil
}

// memLayer is an in-memory path to tar header map for one image layer.
type memLayer struct {
	files map[string]memFile // Path to memFile map
}

// newMemLayer inits a new memLayer instance.
func newMemLayer() *memLayer {
	return &memLayer{
		files: make(map[string]memFile),
	}
}

// count returns number of files in the layer.
func (l *memLayer) count() int {
	return len(l.files)
}

// createHeader creates a new tar header from given path and file info.
func (l *memLayer) createHeader(root, src, dst string, fi os.FileInfo) (*tar.Header, error) {
	hdr, err := tar.FileInfoHeader(fi, "")
	if err != nil {
		return nil, fmt.Errorf("create header %s: %s", src, err)
	}

	// Set name. FileInfoHeader only set name to file base name by default.
	// Also remove leading "/" in dst. Tars produced by docker don't have it.
	hdr.Name = pathutils.RelPath(dst)
	hdr.Uname = ""
	hdr.Gname = ""

	src = pathutils.AbsPath(src)

	switch hdr.Typeflag {
	case tar.TypeDir:
		// Directories in docker generated layers has trailing slashes.
		if !strings.HasSuffix(src, "/") {
			hdr.Name += "/"
		}
	case tar.TypeSymlink:
		if ok, target, err := resolveSymlink(src, fi); err != nil {
			return nil, fmt.Errorf("resolve symlink %s: %s", src, err)
		} else if !ok {
			return nil, fmt.Errorf("symlink in tar header but not on disk: %s", src)
		} else {
			if filepath.IsAbs(target) {
				target, err = pathutils.TrimRoot(target, root)
				if err != nil {
					return nil, fmt.Errorf("trim symlink root: %s", err)
				}
			}
			hdr.Linkname = target
		}
	case tar.TypeLink, tar.TypeReg, tar.TypeRegA:
		// TODO: Handle hard link detection here.
	}
	return hdr, nil
}

// addHeader adds given tar header to layer.
// If the given path has whiteout prefix, path of the deleted file/dir will be
// used as key.
// Otherwise, a valid src path should be provided, where the content could be
// read from when creating tar, and that might not be the same path as dst.
func (l *memLayer) addHeader(src, dst string, hdr *tar.Header) memFile {
	src = pathutils.AbsPath(src)
	dst = pathutils.AbsPath(dst)
	d, b := filepath.Split(dst)

	var mf memFile
	if strings.HasPrefix(b, _whiteoutPrefix) {
		deleted := d + strings.TrimPrefix(b, _whiteoutPrefix)
		mf = newWhiteoutMemFile(deleted, dst)
		l.files[deleted] = mf
	} else {
		mf = newContentMemFile(src, dst, hdr)
		l.files[dst] = mf
	}
	return mf
}

// addWhiteout adds a whiteout file for a file/dir to be removed.
// Path of the file/dir to be deleted will be used as key.
// Note: The given path shouldn't contain whiteout prefix.
func (l *memLayer) addWhiteout(p string) (memFile, error) {
	d, b := filepath.Split(pathutils.AbsPath(p))

	if strings.HasPrefix(b, _whiteoutPrefix) {
		return nil, fmt.Errorf("base name contains whiteout prefix: %s", p)
	}

	whiteoutPath := path.Join(d, fmt.Sprintf("%s%s", _whiteoutPrefix, b))
	mf := newWhiteoutMemFile(p, whiteoutPath)
	l.files[p] = mf
	return mf, nil
}

// range sort all files and iterate through them with given function.
// TODO: loaded tars normally have files sorted already: avoid unnecessary work.
func (l *memLayer) rangeFiles(f func(memFile) error) error {
	keys := make([]string, 0)
	for p := range l.files {
		keys = append(keys, p)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if err := f(l.files[key]); err != nil {
			return fmt.Errorf("apply f to %s: %s", key, err)
		}
	}
	return nil
}
