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
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUpdateFileInfo(t *testing.T) {
	t.Run("UpdateDir", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		testDir1, err := ioutil.TempDir(tmpRoot, "test1")
		require.NoError(err)
		testDir2, err := ioutil.TempDir(tmpRoot, "test2")
		require.NoError(err)

		os.Chmod(testDir1, 0755)
		atime := time.Now().Add(-time.Hour * 2)
		mtime := time.Now().Add(-time.Hour)
		os.Chtimes(testDir1, atime, mtime)

		fi1, err := os.Lstat(testDir1)
		require.NoError(err)
		fi2, err := os.Lstat(testDir2)
		require.NoError(err)
		require.NotEqual(fi1.Mode(), fi2.Mode())
		require.NotEqual(fi1.ModTime(), fi2.ModTime())
		require.True(fi1.IsDir())
		require.True(fi2.IsDir())

		header, err := tar.FileInfoHeader(fi1, "")
		require.NoError(err)
		require.NoError(ApplyHeader(testDir2, header))

		fi2, err = os.Stat(testDir2)
		require.NoError(err)
		require.Equal(fi1.Mode(), fi2.Mode())
		require.Equal(fi1.ModTime(), fi2.ModTime())
		require.True(fi1.IsDir())
		require.True(fi2.IsDir())
	})

	t.Run("UpdateFile", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		testFile1, err := ioutil.TempFile(tmpRoot, "test1")
		require.NoError(err)
		testFile2, err := ioutil.TempFile(tmpRoot, "test2")
		require.NoError(err)

		os.Truncate(testFile1.Name(), 100)
		os.Chmod(testFile1.Name(), 0755)
		atime := time.Now().Add(-time.Hour)
		mtime := time.Now().Add(-time.Hour * 2)
		os.Chtimes(testFile1.Name(), atime, mtime)

		fi1, err := os.Lstat(testFile1.Name())
		require.NoError(err)
		fi2, err := os.Lstat(testFile2.Name())
		require.NoError(err)
		require.NotEqual(fi1.Mode(), fi2.Mode())
		require.NotEqual(fi1.ModTime(), fi2.ModTime())
		require.NotEqual(fi1.Size(), fi2.Size())
		require.False(fi1.IsDir())
		require.False(fi2.IsDir())

		header, err := tar.FileInfoHeader(fi1, "")
		require.NoError(err)
		require.NoError(ApplyHeader(testFile2.Name(), header))

		fi2, err = os.Stat(testFile2.Name())
		require.NoError(err)
		require.Equal(fi1.Mode(), fi2.Mode())
		require.Equal(fi1.ModTime(), fi2.ModTime())
		require.NotEqual(fi1.Size(), fi2.Size()) // Sizes should still be different.
		require.False(fi1.IsDir())
		require.False(fi2.IsDir())
	})

	t.Run("UpdateSymlinkTriggersError", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		testFile1, err := ioutil.TempFile(tmpRoot, "test1")
		require.NoError(err)
		testLink2 := path.Join(tmpRoot, "test_link")
		require.NoError(os.Symlink(testFile1.Name(), testLink2))

		fi1, err := os.Lstat(testFile1.Name())
		require.NoError(err)
		header, err := tar.FileInfoHeader(fi1, "")
		require.NoError(err)
		require.Error(ApplyHeader(testLink2, header))
	})

	t.Run("UpdateFileWithSymlinkHeaderTriggersError", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		testFile1, err := ioutil.TempFile(tmpRoot, "test1")
		require.NoError(err)
		testLink2 := path.Join(tmpRoot, "test_link")
		require.NoError(os.Symlink(testFile1.Name(), testLink2))

		fi2, err := os.Lstat(testLink2)
		require.NoError(err)
		header, err := tar.FileInfoHeader(fi2, "")
		require.NoError(err)
		require.Error(ApplyHeader(testFile1.Name(), header))
	})
}
