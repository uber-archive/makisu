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
// locations. Common behaviors regardless of the parameters:
// - If target diretory's ancestors do not exist, they will be created with
//   default 0755 permission and owned by root.
// - If target directory already exists, its permission and owner will be
//   preserved.
// - Symlinks are copied with original targets (not guaranteed to be valid).
//
// Then there are 3 scenarios for handling file/directory permissions:
// - ADD/COPY without flags:
//   - If target directory doesn't exist, it will be created with default 0755
//     permission and owned by root.
//   - For directories and files under source, permissions and owners will be
//     preserved.
//   - Leave dstDirOwner and dstFileAndChildrenOwner empty
// - ADD/COPY --chown:
//   - If target directory doesn't exist, it will be created with default 0755
//     permission and owned by given uid/gid.
//   - For directories and files under source, permissions are kept, but owner
//     will be changed to given uid/gid.
//   - Use dstDirOwner without overwrite and dstFileAndChildrenOwner with
//     overwrite.
// - ADD/COPY --archive:
//   - If target directory doesn't exist, it will be created with default 0755
//     permission and owned by source dir's uid/gid (directly given as copier
//     parameters).
//   - For directories and files under source, permissions and owners will be
//     preserved.
//   - Use dstDirOwner without overwrite.
type Copier struct {
	blacklist []string

	dstDirOwner             *Owner
	dstFileAndChildrenOwner *Owner
}

// Owner is a tuple of uid+gid, and a flag to indicate whether to overwrite
// existing owner.
type Owner struct {
	uid       int
	gid       int
	overwrite bool
}

// NewCopier initializes a new copier object. Files from provided blacklist will
// be ignored.
func NewCopier(blacklist []string, opts ...CopyOption) *Copier {
	c := &Copier{
		blacklist: append(blacklist),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type CopyOption func(*Copier)

func WithDstDirOwner(uid, gid int, overwrite bool) CopyOption {
	return func(c *Copier) {
		c.dstDirOwner = &Owner{
			uid:       uid,
			gid:       gid,
			overwrite: overwrite,
		}
	}
}

func WithDstFileAndChildrenOwner(uid, gid int, overwrite bool) CopyOption {
	return func(c *Copier) {
		c.dstFileAndChildrenOwner = &Owner{
			uid:       uid,
			gid:       gid,
			overwrite: overwrite,
		}
	}
}

// CopyFile copies the content and permissions of the file at src to dst.
// If the target file exists, its contents and permissions might be overwritten,
// depending on copier attributes.
// For symlinks, the link target will be copied as-is.
func (c *Copier) CopyFile(source, target string) error {
	// Make target parent directories (uid and gid will be computed from the sources one).
	targetDir := filepath.Dir(target)
	if err := c.mkdirAll(targetDir); err != nil {
		return fmt.Errorf("mkdir all %s: %s", targetDir, err)
	}
	// Copy file permissions and contents.
	return c.copyFile(source, target)
}

// CopyDir recursively copies the directory at source to target. The source
// directory must exist, and the target doesn't need to exist but must be a
// directory if it does. Existing children files/dirs will be overwritten.
// Permissions and owners depend on copier attributes.
//
// Note: If src contains dst, this function would break infinite loop silently.
// This is needed to defend against scenarios like:
//   COPY --from=stage1 / /
// where / will be stashed to some child directory at the end of stage1, and
// causes infinite loop.
func (c *Copier) CopyDir(source, target string) error {
	if c.isBlacklisted(source) {
		// Ignore this directory since it's blacklisted.
		log.Infof("* Ignoring copy of directory %s because it is blacklisted", source)
		return nil
	}

	// Make target directory and parent directories.
	if err := c.mkdirAll(target); err != nil {
		return fmt.Errorf("mkdir all %s: %s", target, err)
	}

	// Recursively copy contents of source directory.
	return c.copyDirContents(source, target, target)
}

func (c *Copier) isBlacklisted(source string) bool {
	return pathutils.IsDescendantOfAny(source, c.blacklist)
}

// copyFile copies the permissions and contents of the file at src to dst.
func (c *Copier) copyFile(src, dst string) error {
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

	// Handle regular files.
	return c.copyRegularFile(fi, src, dst)
}

// Open both files, creating dst if need be.
func (c *Copier) copyRegularFile(fi os.FileInfo, src, dst string) error {
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

	// Change the mode of dst to that of src, and update owner accordingly.
	// Note: Chmod needs to be called after chown, otherwise setuid and setgid
	// bits could be unset.
	uid, gid := fileOwners(fi)
	if c.dstFileAndChildrenOwner != nil && c.dstFileAndChildrenOwner.overwrite {
		uid = c.dstFileAndChildrenOwner.uid
		gid = c.dstFileAndChildrenOwner.gid
	}
	if err := os.Chown(dst, uid, gid); err != nil {
		return fmt.Errorf("chown %s: %s", dst, err)
	}
	if err := os.Chmod(dst, fi.Mode()); err != nil {
		return fmt.Errorf("chmod %s: %s", dst, err)
	}
	return nil
}

func (c *Copier) copySymlink(src, dst string) error {
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

// copyDirContents recursively copies the contents of directory src to dst.
// Both must exist.
func (c *Copier) copyDirContents(src, dst, origDst string) error {
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
			if err := c.copyChildDir(currSrc, currDst); err != nil {
				return fmt.Errorf("copy dir %s to %s: %s", currSrc, currDst, err)
			}
			if err := c.copyDirContents(currSrc, currDst, origDst); err != nil {
				return fmt.Errorf("copy dir contents %s to %s: %s", currSrc, currDst, err)
			}
		} else {
			if err := c.copyFile(currSrc, currDst); err != nil {
				return fmt.Errorf("copy file %s to %s: %s", currSrc, currDst, err)
			}
		}
	}
	return nil
}

// copyDir copies the directory at src to dst.
func (c *Copier) copyChildDir(src, dst string) error {
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

	// Create the dst directory with src's mode if it doesn't exist, else chmod it
	// to the same.
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

	// Change mode of dst to that of src, and change owner of dst accordingly.
	// Note: Chmod needs to be called after chown, otherwise setuid and setgid
	// bits could be unset.
	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("chmod %s: %s", dst, err)
	}
	uid, gid := fileOwners(srcInfo)
	if c.dstFileAndChildrenOwner != nil && c.dstFileAndChildrenOwner.overwrite {
		uid = c.dstFileAndChildrenOwner.uid
		gid = c.dstFileAndChildrenOwner.gid
	}
	if err := os.Chown(dst, uid, gid); err != nil {
		return fmt.Errorf("chown %s: %s", dst, err)
	}

	return nil
}

// mkdirAll performs the same operation as os.MkdirAll, except it also changes
// permission and owner of dst directory.
// - Parent directories will have 0755 permission and owner by root.
// - Target directory's permission and owner will be decided by copier
//   attributes.
func (c *Copier) mkdirAll(dst string) error {
	if dst == "" {
		return errors.New("empty dst directory")
	}
	abs, err := filepath.Abs(filepath.Clean(dst))
	if err != nil {
		return fmt.Errorf("failed to get absolute path of %s: %s", dst, err)
	}

	split := strings.Split(abs, "/")
	split[0] = "/"
	var prevDir string
	for i, dir := range split {
		if i == len(split)-1 {
			break
		}

		currDir := filepath.Join(prevDir, dir)
		if _, err := os.Lstat(currDir); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("stat %s: %s", currDir, err)
			}

			// Create dir with default mode and owner.
			if err := os.Mkdir(currDir, 0755); err != nil {
				return fmt.Errorf("mkdir %s with default mode 0755: %s", currDir, err)
			}
			if err := os.Chown(currDir, 0, 0); err != nil {
				return fmt.Errorf("chown %s with default owner (0:0): %s", currDir, err)
			}
		}
		prevDir = currDir
	}

	if _, err := os.Lstat(abs); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("stat %s: %s", abs, err)
		}
		// Create dst dir with default mode and specified owner.
		if err := os.Mkdir(abs, 0755); err != nil {
			return fmt.Errorf("mkdir %s with default mode 0755: %s", abs, err)
		}

		if c.dstDirOwner != nil {
			if err := os.Chown(abs, c.dstDirOwner.uid, c.dstDirOwner.gid); err != nil {
				return fmt.Errorf(
					"chown %s with owner (%d:%d): %s", abs, c.dstDirOwner.uid, c.dstDirOwner.gid, err)
			}
		} else {
			if err := os.Chown(abs, 0, 0); err != nil {
				return fmt.Errorf("chown %s with default owner (0:0): %s", abs, err)
			}
		}
	} else {
		if c.dstDirOwner != nil && c.dstDirOwner.overwrite {
			// Change dst dir with specified owner.
			if err := os.Chown(abs, c.dstDirOwner.uid, c.dstDirOwner.gid); err != nil {
				return fmt.Errorf(
					"chown %s with owner (%d:%d): %s", abs, c.dstDirOwner.uid, c.dstDirOwner.gid, err)
			}
		}
	}

	return nil
}
