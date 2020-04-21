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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/mountutils"
	"github.com/uber/makisu/lib/pathutils"
	"github.com/uber/makisu/lib/tario"
	"github.com/uber/makisu/lib/utils"
)

// shouldSkip returns true if the path is a descendent of any path in the blacklist,
// a special file, or a mount point.
func shouldSkip(path string, fi os.FileInfo, blacklist []string) (bool, error) {
	if strings.HasPrefix(filepath.Base(path), _whiteoutMetaPrefix) {
		// If it's a AUFS metadata file or dir, simply ignore.
		// TODO: There could be hardlinks pointing to files under /.wh..wh.plnk.
		// Taking the simplest solution for now, but this is preventing us from
		// deduping hardlinks.
		return true, nil
	} else if pathutils.IsDescendantOfAny(path, blacklist) || (fi != nil && utils.IsSpecialFile(fi)) {
		return true, nil
	} else if isMountpoint, err := mountutils.IsMountpoint(path); err != nil {
		return false, fmt.Errorf("check mount point: %s", err)
	} else if isMountpoint {
		return true, nil
	}
	return false, nil
}

func walk(srcRoot string, blacklist []string, f func(string, os.FileInfo) error) error {
	if err := filepath.Walk(srcRoot, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("starting walk %s: %s", p, err)
		} else if skip, err := shouldSkip(p, fi, blacklist); err != nil {
			return fmt.Errorf("check should skip: %s", err)
		} else if skip {
			if fi.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if err := f(p, fi); err != nil {
			return fmt.Errorf("applying f to %s: %s", p, err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("walking %s: %s", srcRoot, err)
	}
	return nil
}

// removePathRecursive attempts to recursively remove everything under the given path,
// excluding paths specified by the blacklist. Returns true if it succeeds in removing
// everything under the path.
func removePathRecursive(p string, fi os.FileInfo, blacklist []string) bool {
	if skip, err := shouldSkip(p, fi, blacklist); err != nil {
		log.Errorf("failed to check if should skip %s: %s", p, err)
		return false
	} else if skip {
		return false
	}

	// For non-directories, just attempt to remove the file.
	if !fi.IsDir() {
		if err := os.Remove(p); err != nil {
			log.Errorf("failed to remove file %s: %s", p, err)
			return false
		}
		return true
	}

	// For directories, recursively remove all children. If all succeed,
	// remove the directory itself.
	var anyFailed bool
	children, err := ioutil.ReadDir(p)
	if err != nil {
		log.Errorf("failed to read dir %s: %s", p, err)
		return false
	}
	for _, fi := range children {
		if !removePathRecursive(filepath.Join(p, fi.Name()), fi, blacklist) {
			anyFailed = true
		}
	}
	if anyFailed {
		return false
	}
	if os.RemoveAll(p) != nil {
		log.Errorf("failed to remove directory: %s", p)
		return false
	}
	return true
}

// removeAllChildren recursively removes all of the files that it can under the given root.
// It skips paths in the given blacklist and continues when it fails to remove a file.
func removeAllChildren(srcRoot string, blacklist []string) error {
	children, err := ioutil.ReadDir(srcRoot)
	if err != nil {
		return fmt.Errorf("failed to get children of %s: %s", srcRoot, err)
	}
	for _, child := range children {
		removePathRecursive(filepath.Join(srcRoot, child.Name()), child, blacklist)
	}
	return nil
}

// resolveHardLink linked inode the the given path.
// For docker's implementation, see:
//   https://github.com/moby/moby/blob/master/pkg/archive/archive.go
func resolveHardLink(p string, fi os.FileInfo) uint64 {
	return uint64(utils.FileInfoStat(fi).Ino)
}

// resolveSymlink returns true and link target if the given path is a symlink.
func resolveSymlink(p string, fi os.FileInfo) (bool, string, error) {
	if fi.Mode()&os.ModeSymlink == 0 {
		return false, "", nil
	}
	target, err := os.Readlink(p)
	if err != nil {
		return false, "", fmt.Errorf("read link: %s", err)
	}
	return true, target, nil
}

// CreateTarFromDirectory creates a tar archive containing the contents of the given
// directory. It also compresses the contents with given compression level.
func CreateTarFromDirectory(target, dir string) error {
	file, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("open target file: %s", err)
	}
	defer file.Close()

	var tw *tar.Writer
	gw, err := tario.NewGzipWriter(file)
	if err != nil {
		return fmt.Errorf("new gzip writer: %s", err)
	}
	defer gw.Close()
	tw = tar.NewWriter(gw)
	defer tw.Close()

	inodes := make(map[uint64]string)
	return filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walk: %s", err)
		}
		if dir == p {
			return nil
		}
		return tarOneItem(dir, p, fi, tw, inodes)
	})
}

// tarOneItem writes the header and (optionally) data corresponding to p to the tar writer.
func tarOneItem(root, p string, fi os.FileInfo, tw *tar.Writer, inodes map[uint64]string) error {
	var err error
	link := fi.Name()
	if fi.Mode()&os.ModeSymlink != 0 {
		link, err = os.Readlink(p)
		if err != nil {
			return fmt.Errorf("read link: %s", err)
		}
		link, err = pathutils.TrimRoot(link, root)
		if err != nil {
			return fmt.Errorf("trim link: %s", err)
		}
	}
	hdr, err := tar.FileInfoHeader(fi, link)
	if err != nil {
		return fmt.Errorf("file info header: %s", err)
	}
	trimmed, err := pathutils.TrimRoot(p, root)
	if err != nil {
		return fmt.Errorf("trim root: %s", err)
	}
	hdr.Name = pathutils.RelPath(trimmed)
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("write header: %s", err)
	}

	// Note: For hard links and regular files, if it points to an inode this
	// layer hasn't seen before, it will be treated as a regular file.
	// Subsequent hard links and regular files pointing to the same inode
	// will all be treated as hard links.
	if hdr.Typeflag == tar.TypeLink || hdr.Typeflag == tar.TypeReg ||
		hdr.Typeflag == tar.TypeRegA {

		inode := resolveHardLink(p, fi)
		if target, ok := inodes[inode]; ok {
			hdr.Typeflag = tar.TypeLink
			hdr.Size = 0
			hdr.Linkname = target
		} else {
			inodes[inode] = hdr.Name
		}
	}

	// Copy file content for regular files only.
	if fi.Mode().IsRegular() {
		f, err := os.Open(p)
		if err != nil {
			return fmt.Errorf("open f: %s", err)
		}
		defer f.Close()
		if _, err := io.Copy(tw, f); err != nil {
			return fmt.Errorf("write file content: %s", err)
		}
	}
	return nil
}

// evalSymlinks returns the path name after the evaluation of any symbolic links.
// When actually operating on files, joins their absolute paths to srcRoot. This
// function assumes that the path passed corresponds to a file that exists and that
// is a symlink.
//
// This function and the below helpers are minimally modified from the
// filepath.EvalSymlinks source at: https://golang.org/src/path/filepath/symlink.go
func evalSymlinks(p, srcRoot string) (string, error) {
	if p == "" {
		return p, nil
	}
	var linksWalked int // to protect against cycles
	for {
		i := linksWalked
		newpath, err := walkLinks(p, srcRoot, &linksWalked)
		if err != nil {
			return "", err
		}
		if i == linksWalked {
			return pathutils.AbsPath(newpath), nil
		}
		p = newpath
	}
}

func walkLink(path, root string, linksWalked *int) (newpath string, islink bool, err error) {
	if *linksWalked > 255 {
		return "", false, errors.New("eval symlinks: too many links")
	}
	fi, err := os.Lstat(filepath.Join(root, path))
	if err != nil {
		return "", false, fmt.Errorf("lstat: %s", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		return path, false, nil
	}
	newpath, err = os.Readlink(filepath.Join(root, path))
	if err != nil {
		return "", false, err
	} else if !filepath.HasPrefix(newpath, root) && filepath.IsAbs(newpath) {
		return "", false, fmt.Errorf("link points outside of root: %s -> %s", filepath.Join(root, path), newpath)
	}
	*linksWalked++
	newpath = strings.TrimPrefix(newpath, root)
	return newpath, true, nil
}

func walkLinks(path, root string, linksWalked *int) (string, error) {
	switch dir, file := filepath.Split(path); {
	case dir == "":
		newpath, _, err := walkLink(file, root, linksWalked)
		if err != nil {
			return newpath, fmt.Errorf("walk link: %s", err)
		}
		return newpath, nil
	case file == "":
		if os.IsPathSeparator(dir[len(dir)-1]) {
			if strings.TrimRight(dir, "/") == strings.TrimRight(root, "/") {
				return dir, nil
			}
			return walkLinks(dir[:len(dir)-1], root, linksWalked)
		}
		// TODO(pourchet): confirm unreachable code?
		newpath, _, err := walkLink(dir, root, linksWalked)
		return newpath, err
	default:
		newdir, err := walkLinks(dir, root, linksWalked)
		if err != nil {
			return "", err
		}
		newpath, islink, err := walkLink(filepath.Join(newdir, file), root, linksWalked)
		if err != nil {
			return "", fmt.Errorf("walk link: %s", err)
		}
		if !islink {
			return newpath, nil
		}
		if filepath.IsAbs(newpath) || os.IsPathSeparator(newpath[0]) {
			return newpath, nil
		}
		return filepath.Join(newdir, newpath), nil
	}
}

// compare tar files of memFSNodes.
func compareNode(node1, node2 *memFSNode, missing1, missing2 map[string]*memFSNode, diff1, diff2 map[string]*tar.Header, path string) {
	// some issue with the IsSimilarHeader, I remove the time check.
	if isSimilar, _ := tario.IsSimilarHeader(node1.hdr, node2.hdr); !isSimilar {
		diff1[path] = node1.hdr
		diff2[path] = node2.hdr
	}

	allChildren := make(map[string]bool)
	for child := range node1.children {
		allChildren[child] = true
	}

	for child := range node2.children {
		allChildren[child] = true
	}

	for child := range allChildren {
		nextNode1, ok1 := node1.children[child]
		nextNode2, ok2 := node2.children[child]
		updatedPath := filepath.Join(path, child)
		if ok1 && ok2 {
			compareNode(nextNode1, nextNode2, missing1, missing2, diff1, diff2, updatedPath)
			continue
		} else if ok1 {
			missing2[updatedPath] = node1.children[child]
		} else if ok2 {
			missing1[updatedPath] = node2.children[child]
		}
	}
}

// log the info of a memFSNode.
func checkMemFsNode(node *memFSNode) {
	hdr := node.hdr
	log.Infof("%s %s %d %d %d", hdr.Name, hdr.FileInfo().Mode(), hdr.Uid, hdr.Gid, hdr.Size)
	for _, nxtNode := range node.children {
		checkMemFsNode(nxtNode)
	}
}

func CompareFS(fs1, fs2 *MemFS, image1, image2 image.Name) {
	missing1 := make(map[string]*memFSNode)
	missing2 := make(map[string]*memFSNode)

	diff1 := make(map[string]*tar.Header)
	diff2 := make(map[string]*tar.Header)

	compareNode(fs1.tree, fs2.tree, missing1, missing2, diff1, diff2, "")
	image1Format := image1.GetRepository() + ":" + image1.GetTag()
	image2Format := image2.GetRepository() + ":" + image2.GetTag()
	// Files missing in first image but appeared in second image
	log.Infof("======== file missing in first image %s=====", image1Format)
	for _, node := range missing2 {
		checkMemFsNode(node)
	}

	// Files missing in second image but appeared in first image.
	log.Infof("========= file missing in second image %s=====", image2Format)
	for _, node := range missing1 {
		checkMemFsNode(node)
	}

	// File differences in two images.
	log.Infof("======== difference between two images %s and %s ======", image1Format, image2Format)
	for path := range diff1 {
		hdr1, hdr2 := diff1[path], diff2[path]
		log.Infof("%s %s %d %d %d %s %d %d %d", path, hdr1.FileInfo().Mode(), hdr1.Uid, hdr1.Gid, hdr1.Size, hdr2.FileInfo().Mode(), hdr2.Uid, hdr2.Gid, hdr2.Size)
	}
}
