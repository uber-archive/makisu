package tario

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// WriteEntry write the file from the local filesystem into the tar writer.
// This function doesn't handle parent directories.
func WriteEntry(w *tar.Writer, src string, h *tar.Header) error {
	if err := WriteHeader(w, h); err != nil {
		return fmt.Errorf("write header helper: %v", err)
	}

	switch h.Typeflag {
	case tar.TypeDir, tar.TypeLink, tar.TypeSymlink:
		return nil
	case tar.TypeReg, tar.TypeRegA:
		f, err := os.Open(src)
		if err != nil {
			return fmt.Errorf("open src file %s: %s", src, err)
		}
		defer f.Close()

		// Using CopyN here because there could be dangling process still
		// writing to the file at the time size is collected.
		if _, err := io.CopyN(w, f, h.Size); err != nil {
			return fmt.Errorf("copy file %s to tar writer: %s", src, err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported type %b", h.Typeflag)
	}
}

// WriteHeader writes the header given to the tar writer.
func WriteHeader(w *tar.Writer, h *tar.Header) error {
	// Remove leading "/" in dst. Tars produced by docker doesn't have it.
	h.Name = strings.TrimLeft(h.Name, "/")

	// Golang by default _rounds_ the modtime before writing the tar header, but
	// the GNU tar program _truncates_ that modtime. Manually truncate the time
	// to avoid inconsistency.
	h.ModTime = h.ModTime.Truncate(1 * time.Second)

	if err := w.WriteHeader(h); err != nil {
		return fmt.Errorf("write header %s: %s", h.Name, err)
	}
	return nil
}
