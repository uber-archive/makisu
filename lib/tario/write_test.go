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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteEntry(t *testing.T) {
	t.Run("WriteDirectory", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)
		outRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(outRoot)

		// Create directory.
		d, err := ioutil.TempDir(tmpRoot, "test")
		require.NoError(err)
		require.NoError(os.Chmod(d, 0777))

		// Create tar.
		fi, err := os.Lstat(d)
		require.NoError(err)
		h, err := tar.FileInfoHeader(fi, "")
		require.NoError(err)
		h.Name = d // FileInfoHeader only set name to file base name by default

		tarFile, err := ioutil.TempFile(tmpRoot, "test.tar")
		require.NoError(err)
		w := tar.NewWriter(tarFile)
		require.NoError(WriteEntry(w, d, h))
		w.Close()

		// Verify.
		require.NoError(untarHelper(tarFile.Name(), outRoot))

		fi, err = os.Lstat(filepath.Join(outRoot, d))
		require.NoError(err)
		require.Equal(uint32(0777), uint32(fi.Mode().Perm()))
	})

	t.Run("WriteHardLink", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)
		outRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(outRoot)

		// Create file and hard link.
		f, err := ioutil.TempFile(tmpRoot, "test")
		require.NoError(err)
		_, err = f.Write([]byte("test data"))
		require.NoError(err)
		f.Close()
		require.NoError(os.Chmod(f.Name(), 0777))
		link := filepath.Join(tmpRoot, "link")
		os.Link(f.Name(), link)

		// Create tar.
		fi, err := os.Lstat(link)
		require.NoError(err)
		h, err := tar.FileInfoHeader(fi, "")
		require.NoError(err)
		h.Name = link // FileInfoHeader only set name to file base name by default
		h.Typeflag = tar.TypeLink
		h.Size = 0
		h.Linkname = strings.TrimLeft(f.Name(), "/") // Link to relative path

		tarFile, err := ioutil.TempFile(tmpRoot, "test.tar")
		require.NoError(err)
		w := tar.NewWriter(tarFile)
		require.NoError(WriteEntry(w, f.Name(), h))
		w.Close()

		// Copy link target to output dir to avoid dangling link.
		require.NoError(os.MkdirAll(filepath.Dir(filepath.Join(outRoot, f.Name())), 0755))
		require.NoError(os.Rename(f.Name(), filepath.Join(outRoot, f.Name())))

		// Verify.
		require.NoError(untarHelper(tarFile.Name(), outRoot))

		fi, err = os.Lstat(filepath.Join(outRoot, link))
		require.NoError(err)
		require.Equal(uint32(0777), uint32(fi.Mode().Perm()))

		b, err := ioutil.ReadFile(filepath.Join(outRoot, link))
		require.NoError(err)
		require.Equal([]byte("test data"), b)

		targetFi, err := os.Lstat(filepath.Join(outRoot, f.Name()))
		require.NoError(err)
		require.Equal(
			uint64(targetFi.Sys().(*syscall.Stat_t).Ino), uint64(fi.Sys().(*syscall.Stat_t).Ino))
		require.True(uint32(fi.Sys().(*syscall.Stat_t).Nlink) > 1)
	})

	t.Run("WriteSymlink", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)
		outRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(outRoot)

		// Create file and hard link.
		f, err := ioutil.TempFile(tmpRoot, "test")
		require.NoError(err)
		_, err = f.Write([]byte("test data"))
		require.NoError(err)
		f.Close()
		require.NoError(os.Chmod(f.Name(), 0777))
		link := filepath.Join(tmpRoot, "link")
		os.Symlink(f.Name(), link)

		// Create tar.
		fi, err := os.Lstat(link)
		require.NoError(err)
		h, err := tar.FileInfoHeader(fi, "")
		require.NoError(err)
		h.Name = link // FileInfoHeader only set name to file base name by default

		h.Linkname = strings.TrimLeft(f.Name(), "/") // Link to relative path

		tarFile, err := ioutil.TempFile(tmpRoot, "test.tar")
		require.NoError(err)
		w := tar.NewWriter(tarFile)
		require.NoError(WriteEntry(w, f.Name(), h))
		w.Close()

		// Copy link target to output dir to avoid dangling link.
		require.NoError(os.MkdirAll(filepath.Dir(filepath.Join(outRoot, f.Name())), 0755))
		require.NoError(os.Rename(f.Name(), filepath.Join(outRoot, f.Name())))

		// Verify.
		require.NoError(untarHelper(tarFile.Name(), outRoot))

		target, err := os.Readlink(filepath.Join(outRoot, link))
		require.NoError(err)
		require.Equal(target, strings.TrimLeft(f.Name(), "/"))
	})

	t.Run("WriteRegularFile", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)
		outRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(outRoot)

		// Create file.
		f, err := ioutil.TempFile(tmpRoot, "test")
		require.NoError(err)
		_, err = f.Write([]byte("test data"))
		require.NoError(err)
		f.Close()
		require.NoError(os.Chmod(f.Name(), 0777))

		// Create tar.
		fi, err := os.Lstat(f.Name())
		require.NoError(err)
		h, err := tar.FileInfoHeader(fi, "")
		require.NoError(err)
		h.Name = f.Name() // FileInfoHeader only set name to file base name by default

		tarFile, err := ioutil.TempFile(tmpRoot, "test.tar")
		require.NoError(err)
		w := tar.NewWriter(tarFile)
		require.NoError(WriteEntry(w, f.Name(), h))
		w.Close()

		// Verify.
		require.NoError(untarHelper(tarFile.Name(), outRoot))

		fi, err = os.Lstat(filepath.Join(outRoot, f.Name()))
		require.NoError(err)
		require.Equal(uint32(0777), uint32(fi.Mode().Perm()))

		b, err := ioutil.ReadFile(filepath.Join(outRoot, f.Name()))
		require.NoError(err)
		require.Equal([]byte("test data"), b)
	})
}
