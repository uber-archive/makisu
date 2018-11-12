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
