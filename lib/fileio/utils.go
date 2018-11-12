package fileio

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ReaderToFile copies the data from a reader to a destination file.
func ReaderToFile(r io.Reader, dst string) error {
	dst = filepath.Clean(dst)
	w, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create file %s: %s", dst, err)
	}
	defer w.Close()

	if _, err = io.Copy(w, r); err != nil {
		return fmt.Errorf("copy to file %s: %s", dst, err)
	}
	return nil
}
