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

package fileio

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/pathutils"
	"github.com/uber/makisu/lib/utils"
)

// Copier provides helper functions that copy files and directories to specified
// locations.
type Copier interface {
	CopyFile(source, target string, uid, gid int) error
	CopyDir(source, target string, uid, gid int) error
	CopyDirPreserveOwner(source, target string) error
	CopyFilePreserveOwner(source, target string) error
}

type copier struct {
	blacklist []string
}

// NewCopier initializes a new copier object. Files from provided blacklist will
// be ignored.
func NewCopier(blacklist []string) Copier {
	return &copier{
		blacklist: append(blacklist),
	}
}

// NewInternalCopier initializes a new copier object. It is used for copying
// checkpointed files from sandbox dir, and there is no need to blacklist any
// path, since they would have been filtered out by checkpoint.
func NewInternalCopier() Copier {
	return &copier{}
}

// CopyFile copies the content and permissions of the file at src to dst.
// If the target file exists, its contents and permissions will be replaced.
// If the parent directories of dst do not exist, they will be created with
// the given uid/gid.
// For symlinks, the link target will be copied as-is.
func (c copier) CopyFile(source, target string, uid, gid int) error {
	// Make target parent directories (uid and gid will be computed from the sources one).
	targetDir := filepath.Dir(target)
	if err := mkdirAll(targetDir, os.ModePerm, uid, gid, false); err != nil {
		return fmt.Errorf("mkdir all %s: %s", targetDir, err)
	}
	// Copy file permissions and contents.
	return c.copyFile(source, target, uid, gid, false)
}

// CopyFilePreserveOwner follow the behavior of CopyFile but preserve file rights.
func (c copier) CopyFilePreserveOwner(source, target string) error {
	// Make target parent directories with passed uid & gid if they don't exist.
	targetDir := filepath.Dir(target)
	if err := mkdirAll(targetDir, os.ModePerm, 0, 0, true); err != nil {
		return fmt.Errorf("mkdir all %s: %s", targetDir, err)
	}
	// Copy file permissions and contents.
	return c.copyFile(source, target, 0, 0, true)
}

// CopyDir recursively copies the directory at source to target. The source
// directory must exist, and the target doesn't need to exist but must be a
// directory if it does. If the target or any of its ancestors do not exist,
// they are created with default permissions and owned by the given uid/gid.
// Permissions and ownership are preserved for directories and files under source.
// Symlinks are copied with their original target (not guaranteed to be valid).
//
// If src contains dst, this function would break infinite loop silently.
// This is needed to defend against scenarios like:
//   COPY --from=stage1 / /
// where / will be stashed to some child directory at the end of stage1, and
// causes infinite loop.
func (c copier) CopyDir(source, target string, uid, gid int) error {
	if c.isBlacklisted(source) {
		// Ignore this directory since it's blacklisted.
		log.Infof("* Ignoring copy of directory %s because it is blacklisted", source)
		return nil
	}
	// Make target parent directories with passed uid & gid if they don't exist.
	if err := mkdirAll(target, os.ModePerm, uid, gid, false); err != nil {
		return fmt.Errorf("mkdir all %s: %s", target, err)
	}
	// Recursively copy directories and files.
	return c.copyDirContents(source, target, target, uid, gid, false)
}

// CopyDirPreserveOwner follow the behavior of CopyFile but preserve file rights.
func (c copier) CopyDirPreserveOwner(source, target string) error {
	if c.isBlacklisted(source) {
		// Ignore this directory since it's blacklisted.
		log.Infof("* Ignoring copy of directory %s because it is blacklisted", source)
		return nil
	}
	// Make target parent directories (uid and gid will be computed from the sources one).
	if err := mkdirAll(target, os.ModePerm, 0, 0, true); err != nil {
		return fmt.Errorf("mkdir all %s: %s", target, err)
	}
	// Recursively copy directories and files.
	return c.copyDirContents(source, target, target, 0, 0, true)
}

func (c copier) isBlacklisted(source string) bool {
	return pathutils.IsDescendantOfAny(source, c.blacklist)
}

// copyFile copies the permissions and contents of the file at src to dst.
func (c copier) copyFile(src, dst string, uid, gid int, preserveOwner bool) error {
	fi, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("lstat %s: %s", src, err)
	} else if c.isBlacklisted(src) {
		// Do nothing if this file is blacklisted.
		log.Infof("* Ignoring copy of file %s because it is blacklisted", src)
	} else if utils.IsSpecialFile(fi) {
		// If this is a socket/device/named-pipe, do nothing.
		return nil
	}

	// Handle symlinks.
	// They should not be chown'ed, as chown will change the target's uid/gid.
	if fi.Mode()&os.ModeSymlink != 0 {
		return c.copySymlink(src, dst)
	}

	// If the file already exists, then we will overwrite that file.
	if _, err := os.Lstat(dst); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("lstat %s: %s", dst, err)
	} else if err == nil {
		if err := os.Chmod(dst, os.ModePerm); err != nil {
			return fmt.Errorf("chmod %s: %s", dst, err)
		}
	}

	if preserveOwner {
		uid, gid = fileOwners(fi)
	}

	// Handle regular files.
	return c.copyRegularFile(fi, src, dst, uid, gid)
}

// Open both files, creating dst if need be.
func (c copier) copyRegularFile(fi os.FileInfo, src, dst string, uid, gid int) error {
	r, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s: %s", dst, err)
	}
	defer r.Close()
	w, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		return fmt.Errorf("create %s: %s", dst, err)
	}
	defer w.Close()
	if err := os.Truncate(dst, 0); err != nil {
		return fmt.Errorf("truncate %s: %s", dst, err)
	}

	// Copy contents from src to dst.
	if _, err := io.Copy(w, r); err != nil {
		return fmt.Errorf("copy %s to %s: %s", src, dst, err)
	}

	// Change the owner and mode of dst to that of src.
	// Note: Chmod needs to be called after chown, otherwise setuid and setgid
	// bits could be unset.
	if err := os.Chown(dst, uid, gid); err != nil {
		return fmt.Errorf("chown %s: %s", dst, err)
	}
	if err := os.Chmod(dst, fi.Mode()); err != nil {
		return fmt.Errorf("chmod %s: %s", dst, err)
	}
	return nil
}

func (c copier) copySymlink(src, dst string) error {
	// Remove existing file if path exists.
	if _, err := os.Lstat(dst); err == nil {
		if err := os.Remove(dst); err != nil {
			return fmt.Errorf("remove existing file %s: %s", dst, err)
		}
	}
	// Set symlink target to the original link target.
	linkTarget, err := os.Readlink(src)
	if err != nil {
		return fmt.Errorf("read link %s: %s", src, err)
	}
	if err := os.Symlink(linkTarget, dst); err != nil {
		return fmt.Errorf("write link %s with content %s: %s", dst, linkTarget, err)
	}
	return nil
}

// copyDirContents recursively copies the contents of directory src to dst. Both must exist.
func (c copier) copyDirContents(src, dst, origDst string, uid, gid int, preserveOwner bool) error {
	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return fmt.Errorf("read dir %s: %s", src, err)
	}
	for _, entry := range entries {
		currSrc := filepath.Join(src, entry.Name())
		if c.isBlacklisted(currSrc) {
			// Ignore this directory since it's blacklisted.
			log.Infof("* Ignoring copy of directory %s because it is blacklisted", currSrc)
			continue
		} else if currSrc == origDst {
			// Silently break infinite loop.
			continue
		}
		currDst := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := c.copyDir(currSrc, currDst, uid, gid, preserveOwner); err != nil {
				return fmt.Errorf("copy dir %s to %s: %s", currSrc, currDst, err)
			}
			if err := c.copyDirContents(currSrc, currDst, origDst, uid, gid, preserveOwner); err != nil {
				return fmt.Errorf("copy dir contents %s to %s: %s", currSrc, currDst, err)
			}
		} else {
			if err := c.copyFile(currSrc, currDst, uid, gid, preserveOwner); err != nil {
				return fmt.Errorf("copy file %s to %s: %s", currSrc, currDst, err)
			}
		}
	}
	return nil
}

// copyDir copies the directory at src to dst.
func (c copier) copyDir(src, dst string, uid, gid int, preserveOwner bool) error {
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("lstat %s: %s", src, err)
	} else if !srcInfo.IsDir() {
		return fmt.Errorf("source %s is not a directory", src)
	} else if c.isBlacklisted(src) {
		// Ignore this directory since it's blacklisted.
		log.Infof("* Ignoring copy of directory %s because it is blacklisted", src)
		return nil
	}

	// Make the dst directory with src's mode if it doesn't exist, else chmod it to the same.
	dstInfo, err := os.Lstat(dst)
	if os.IsNotExist(err) {
		if err := os.Mkdir(dst, srcInfo.Mode()); err != nil {
			return fmt.Errorf("mkdir %s: %s", dst, err)
		}
	} else if err != nil {
		return fmt.Errorf("lstat %s: %s", dst, err)
	} else if err == nil && !dstInfo.IsDir() {
		return fmt.Errorf("dst is not a directory")
	}

	// Change owner and mode of dst to that of src.
	// Note: Chmod needs to be called after chown, otherwise setuid and setgid
	// bits could be unset.
	if preserveOwner {
		uid, gid = fileOwners(srcInfo)
	}
	if err := os.Chown(dst, uid, gid); err != nil {
		return fmt.Errorf("chown %s: %s", dst, err)
	}
	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("chmod %s: %s", dst, err)
	}
	return nil
}

// mkdirAll performs the same operation as os.MkdirAll, but also sets the given
// permissions & owners on all created directories.
func mkdirAll(dst string, mode os.FileMode, uid, gid int, preserveOwner bool) error {
	if dst == "" {
		return errors.New("empty target directory")
	}
	abs, err := filepath.Abs(filepath.Clean(dst))
	if err != nil {
		return fmt.Errorf("failed to get absolute path of %s: %s", dst, err)
	}

	split := strings.Split(abs, "/")
	split[0] = "/"

	var prevDir string
	for _, dir := range split {
		absDir := filepath.Join(prevDir, dir)
		if fi, err := os.Lstat(absDir); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("stat %s: %s", absDir, err)
			} else if err := os.Mkdir(absDir, mode); err != nil {
				return fmt.Errorf("mkdir %s: %s", absDir, err)
			}
			if preserveOwner {
				uid, gid = fileOwners(fi)
			}
			if err := os.Chown(absDir, uid, gid); err != nil {
				return fmt.Errorf("chown %s: %s", absDir, err)
			}
		}
		prevDir = absDir
	}
	return nil
}

// fileOwners returns the uid & gid that own the file.
func fileOwners(fi os.FileInfo) (uid int, gid int) {
	stat := utils.FileInfoStat(fi)
	return int(stat.Uid), int(stat.Gid)
}
