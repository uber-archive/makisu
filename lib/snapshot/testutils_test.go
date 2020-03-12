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
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/uber/makisu/lib/tario"

	"github.com/stretchr/testify/require"
)

// merge merges the given layer into MemFS. This is only used for testing, as
// files are merged as the layer is created when adding from a scan/copy.
func (fs *MemFS) merge(l *memLayer) error {
	fs.layers = append(fs.layers, l)

	if err := l.rangeFiles(func(f memFile) error {
		return f.updateMemFS(fs.tree)
	}); err != nil {
		return fmt.Errorf("range files: %s", err)
	}

	return nil
}

func requireEqualLayers(require *require.Assertions, l1 *memLayer, l2 *memLayer) {
	require.Len(l2.files, len(l1.files), "file maps not equal:\nexpected=%+v\nactual=  %+v", l1.files, l2.files)
	for p1, mf1 := range l1.files {
		mf2, ok := l2.files[p1]
		require.True(ok)
		if content1, ok := mf1.(*contentMemFile); ok {
			content2, ok := mf2.(*contentMemFile)
			require.True(ok)
			require.Equal(content1.dst, content2.dst, "%s dsts not equal:\nexpected=%s, actual=%s", p1, content1.dst, content2.dst)
			similar, err := tario.IsSimilarHeader(content1.hdr, content2.hdr)
			require.NoError(err)
			require.True(similar, "%s headers not similar:\nexpected=%+v\nactual=  %+v", p1, content1.hdr, content2.hdr)
		} else if whiteout1, ok := mf1.(*whiteoutMemFile); ok {
			whiteout2, ok := mf2.(*whiteoutMemFile)
			require.True(ok)
			require.Equal(whiteout1.del, whiteout2.del)
			similar, err := tario.IsSimilarHeader(whiteout1.hdr, whiteout2.hdr)
			require.NoError(err)
			require.True(similar)
		} else {
			panic("unknown memfile type")
		}
	}
}

func addDirectoryToLayer(l *memLayer, srcRoot, dst string, perm os.FileMode) error {
	src := filepath.Join(srcRoot, dst)

	// Make sure parent dir modtime is not impacted.
	var parentModTime time.Time
	parentDir := filepath.Dir(src)
	parentFi, err := os.Lstat(parentDir)
	if err == nil {
		parentModTime = parentFi.ModTime()
		defer os.Chtimes(parentDir, parentModTime, parentModTime)
	}

	if err := os.MkdirAll(src, perm); err != nil {
		return fmt.Errorf("mkdir %s: %s", src, err)
	}
	if err := os.Chmod(src, perm); err != nil {
		return fmt.Errorf("chmod %s: %s", src, err)
	}
	fi, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("lstat %s: %s", src, err)
	}

	hdr, err := l.createHeader(srcRoot, src, dst, fi)
	if err != nil {
		return fmt.Errorf("create header %s: %s", dst, err)
	}

	l.addHeader(src, dst, hdr)

	return nil
}

func addRegularFileToLayer(l *memLayer, srcRoot, dst, content string, perm os.FileMode) error {
	src := filepath.Join(srcRoot, dst)

	// Make sure parent dir modtime is not impacted.
	var parentModTime time.Time
	parentDir := filepath.Dir(src)
	parentFi, err := os.Lstat(parentDir)
	if err == nil {
		parentModTime = parentFi.ModTime()
		defer os.Chtimes(parentDir, parentModTime, parentModTime)
	}

	if err := os.Remove(src); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove file %s: %s", src, err)
	}
	f, err := os.OpenFile(src, os.O_WRONLY|os.O_CREATE, perm)
	if err != nil {
		return fmt.Errorf("open file %s: %s", src, err)
	}
	if _, err := f.Write([]byte(content)); err != nil {
		return fmt.Errorf("write to %s: %s", src, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close %s: %s", src, err)
	}
	if err := os.Chmod(src, perm); err != nil {
		return fmt.Errorf("chmod %s: %s", src, err)
	}
	fi, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("lstat %s: %s", src, err)
	}

	hdr, err := l.createHeader(srcRoot, src, dst, fi)
	if err != nil {
		return fmt.Errorf("create header %s: %s", dst, err)
	}

	l.addHeader(src, dst, hdr)
	return nil
}

func addSymlinkToLayer(l *memLayer, srcRoot, dst, target string) error {
	src := filepath.Join(srcRoot, dst)
	target = filepath.Join(srcRoot, target)

	// Make sure parent dir modtime is not impacted.
	var parentModTime time.Time
	parentDir := filepath.Dir(src)
	parentFi, err := os.Lstat(parentDir)
	if err == nil {
		parentModTime = parentFi.ModTime()
		defer os.Chtimes(parentDir, parentModTime, parentModTime)
	}

	if err := os.Symlink(target, src); err != nil {
		return fmt.Errorf("create symlink %s targeting %s: %s", src, target, err)
	}
	fi, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("lstat %s: %s", src, err)
	}

	hdr, err := l.createHeader(srcRoot, src, dst, fi)
	if err != nil {
		return fmt.Errorf("create header %s: %s", dst, err)
	}

	l.addHeader(src, dst, hdr)
	return nil
}

func addHardLinkToLayer(l *memLayer, srcRoot, dst, target string) error {
	src := filepath.Join(srcRoot, dst)
	target = filepath.Join(srcRoot, target)

	// Make sure parent dir modtime is not impacted.
	var parentModTime time.Time
	parentDir := filepath.Dir(src)
	parentFi, err := os.Lstat(parentDir)
	if err == nil {
		parentModTime = parentFi.ModTime()
		defer os.Chtimes(parentDir, parentModTime, parentModTime)
	}

	if err := os.Link(filepath.Join(srcRoot, target), src); err != nil {
		return fmt.Errorf("create hard link %s targeting %s: %s", src, target, err)
	}
	fi, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("lstat %s: %s", src, err)
	}

	hdr, err := l.createHeader(srcRoot, src, dst, fi)
	if err != nil {
		return fmt.Errorf("create header %s: %s", dst, err)
	}

	l.addHeader(src, dst, hdr)
	return nil
}

func addWhiteoutToLayer(l *memLayer, dst string) error {
	if _, err := l.addWhiteout(dst); err != nil {
		return fmt.Errorf("add whiteout %s: %s", dst, err)
	}

	return nil
}

// findNode locates node of given path in mem fs, following symlinks.
// If the path doesn't exist, it would return os.ErrNotExist.
func findNode(fs *MemFS, p string, followSymlink bool, depth int) (*memFSNode, error) {
	if depth >= 1024 {
		return nil, fmt.Errorf("symlink loop at %s", p)
	}

	curr := fs.tree
	parts := strings.Split(strings.Trim(p, "/"), "/")
	for i, part := range parts {
		if n, ok := curr.children[part]; ok {
			if followSymlink && n.hdr.Typeflag == tar.TypeSymlink {
				unresolved := filepath.Join(parts[i+1:]...)
				return findNode(fs, filepath.Join(n.hdr.Linkname, unresolved), true, depth+1)
			}
			if i == len(parts)-1 {
				return n, nil
			}
			curr = n
		} else {
			break
		}
	}

	return nil, os.ErrNotExist
}

func writeTarHelper(m *memFile, srcRoot string, w *tar.Writer) error {
	return filepath.Walk(srcRoot, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if p == srcRoot {
			return nil
		}
		h, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}

		return tario.WriteEntry(w, p, h)
	})
}

func readTarHelper(r *tar.Reader) (map[string]*tar.Header, error) {
	m := make(map[string]*tar.Header)
	for {
		header, err := r.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("read header: %s", err)
		}

		m[header.Name] = header
	}

	return m, nil
}
