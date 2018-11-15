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

package tario

import (
	"archive/tar"
	"fmt"
	"os"
)

// ApplyHeader updates file owner, mtime, and permission bits according to
// header.
// It doesn't change size or type (i.e file to dir).
func ApplyHeader(path string, header *tar.Header) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("lstat %s: %s", path, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 || header.FileInfo().Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("update symlink instead of file: %s", path)
	}

	// Change the owner, mode and mtime of path.
	// Note: Chmod needs to be called after chown, otherwise setuid and setgid
	// bits could be unset.
	if err := os.Chown(path, header.Uid, header.Gid); err != nil {
		return fmt.Errorf("chown %s: %s", path, err)
	}
	if err := os.Chmod(path, header.FileInfo().Mode()); err != nil {
		return fmt.Errorf("chmod %s: %s", path, err)
	}
	mtime := header.FileInfo().ModTime()
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		return fmt.Errorf("chtimes %s: %s", path, err)
	}
	return nil
}
