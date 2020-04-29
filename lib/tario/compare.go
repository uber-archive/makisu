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
	"time"
)

// IsSimilarHeader returns if the given headers are describing similar entries.
func IsSimilarHeader(h *tar.Header, nh *tar.Header, ignoreTime bool) (bool, error) {
	// Don't support modifying "/".
	if h.Name == "" && nh.Name == "" {
		return true, nil
	}

	switch h.Typeflag {
	case tar.TypeSymlink:
		if nh.Typeflag != tar.TypeSymlink {
			return false, nil
		}
		return isSimilarSymlink(h, nh)
	case tar.TypeLink:
		if nh.Typeflag != tar.TypeLink {
			return false, nil
		}
		return isSimilarHardLink(h, nh, ignoreTime)
	case tar.TypeDir:
		if nh.Typeflag != tar.TypeDir {
			return false, nil
		}
		return isSimilarDirectory(h, nh, ignoreTime)
	case tar.TypeReg, tar.TypeRegA:
		if nh.Typeflag != tar.TypeReg && nh.Typeflag != tar.TypeRegA {
			return false, nil
		}
		return isSimilarRegularFile(h, nh, ignoreTime)
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
func isSimilarHardLink(h *tar.Header, nh *tar.Header, ignoreTime bool) (bool, error) {
	timeIsEqual := true
	if !ignoreTime {
		hMtime := h.ModTime.Truncate(1 * time.Second)
		nhMtime := nh.ModTime.Truncate(1 * time.Second)
		timeIsEqual = hMtime.Equal(nhMtime)
	}

	if timeIsEqual &&
		h.Linkname == nh.Linkname &&
		h.Uid == nh.Uid &&
		h.Gid == nh.Gid &&
		h.FileInfo().Mode() == nh.FileInfo().Mode() {
		return true, nil
	}
	return false, nil
}

// isSimilarDirectory returns if the given headers are describing similar
// directories. It only checks mtime and owner, ignoring size, path and content.
func isSimilarDirectory(h *tar.Header, nh *tar.Header, ignoreTime bool) (bool, error) {
	timeIsEqual := true
	if !ignoreTime {
		hMtime := h.ModTime.Truncate(1 * time.Second)
		nhMtime := nh.ModTime.Truncate(1 * time.Second)
		timeIsEqual = hMtime.Equal(nhMtime)
	}

	if timeIsEqual &&
		h.Uid == nh.Uid &&
		h.Gid == nh.Gid &&
		h.FileInfo().Mode() == nh.FileInfo().Mode() {
		return true, nil
	}
	return false, nil
}

// isSimilarRegularFile returns if the given headers are describing similar
// regular files. It only checks mtime, size, and owner, ignoring path and
// content.
func isSimilarRegularFile(h *tar.Header, nh *tar.Header, ignoreTime bool) (bool, error) {
	timeIsEqual := true
	if !ignoreTime {
		hMtime := h.ModTime.Truncate(1 * time.Second)
		nhMtime := nh.ModTime.Truncate(1 * time.Second)
		timeIsEqual = hMtime.Equal(nhMtime)
	}

	if timeIsEqual &&
		h.Uid == nh.Uid &&
		h.Gid == nh.Gid &&
		h.Size == nh.Size &&
		h.FileInfo().Mode() == nh.FileInfo().Mode() {
		return true, nil
	}
	return false, nil
}
