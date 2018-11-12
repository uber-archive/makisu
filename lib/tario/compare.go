package tario

import (
	"archive/tar"
	"fmt"
	"time"
)

// IsSimilarHeader returns if the given headers are describing similar entries.
func IsSimilarHeader(h *tar.Header, nh *tar.Header) (bool, error) {
	if h.Typeflag != nh.Typeflag {
		return false, nil
	}

	// Don't support modifying "/".
	if h.Name == "" && nh.Name == "" {
		return true, nil
	}

	switch h.Typeflag {
	case tar.TypeSymlink:
		return isSimilarSymlink(h, nh)
	case tar.TypeLink:
		return isSimilarHardLink(h, nh)
	case tar.TypeDir:
		return isSimilarDirectory(h, nh)
	case tar.TypeReg, tar.TypeRegA:
		return isSimilarRegularFile(h, nh)
	default:
		return false, fmt.Errorf("unsupported type %b", h.Typeflag)
	}
}

// isSimilarSymlink returns if the given headers are describing similar
// symlinks. It only checks mtime and link target.
func isSimilarSymlink(h *tar.Header, nh *tar.Header) (bool, error) {
	return h.Linkname == nh.Linkname, nil
}

// isSimilarHardLink returns if the given headers are describing similar hard
// links. It only checks mtime and link target.
func isSimilarHardLink(h *tar.Header, nh *tar.Header) (bool, error) {
	hMtime := h.ModTime.Truncate(1 * time.Second)
	nhMtime := nh.ModTime.Truncate(1 * time.Second)
	if hMtime.Equal(nhMtime) &&
		h.Linkname == nh.Linkname &&
		h.Uid == nh.Uid &&
		h.Gid == nh.Gid {
		return true, nil
	}
	return false, nil
}

// isSimilarDirectory returns if the given headers are describing similar
// directories. It only checks mtime and owner, ignoring size, path and content.
func isSimilarDirectory(h *tar.Header, nh *tar.Header) (bool, error) {
	hMtime := h.ModTime.Truncate(1 * time.Second)
	nhMtime := nh.ModTime.Truncate(1 * time.Second)
	if hMtime.Equal(nhMtime) &&
		h.Uid == nh.Uid &&
		h.Gid == nh.Gid {
		return true, nil
	}
	return false, nil
}

// isSimilarRegularFile returns if the given headers are describing similar
// regular files. It only checks mtime, size, and owner, ignoring path and content.
func isSimilarRegularFile(h *tar.Header, nh *tar.Header) (bool, error) {
	hMtime := h.ModTime.Truncate(1 * time.Second)
	nhMtime := nh.ModTime.Truncate(1 * time.Second)
	if hMtime.Equal(nhMtime) &&
		h.Uid == nh.Uid &&
		h.Gid == nh.Gid &&
		h.Size == nh.Size {
		return true, nil
	}
	return false, nil
}
