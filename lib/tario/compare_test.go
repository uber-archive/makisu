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

func TestIsSimilar(t *testing.T) {
	t.Run("DirAndFileConsideredDifferent", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		testDir1, err := ioutil.TempDir(tmpRoot, "test1")
		require.NoError(err)
		testFile2, err := ioutil.TempFile(tmpRoot, "test2")
		require.NoError(err)

		fi1, err := os.Lstat(testDir1)
		require.NoError(err)
		fi2, err := os.Lstat(testFile2.Name())
		require.NoError(err)

		h, err := tar.FileInfoHeader(fi1, "")
		require.NoError(err)
		newH, err := tar.FileInfoHeader(fi2, "")
		require.NoError(err)
		similar, err := IsSimilarHeader(h, newH)
		require.False(similar)
		require.NoError(err)
	})

	t.Run("RootsConsideredSimilar", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		testFile1, err := ioutil.TempFile(tmpRoot, "test1")
		require.NoError(err)
		testFile2, err := ioutil.TempFile(tmpRoot, "test2")
		require.NoError(err)

		fi1, err := os.Lstat(testFile1.Name())
		require.NoError(err)
		fi2, err := os.Lstat(testFile2.Name())
		require.NoError(err)

		h, err := tar.FileInfoHeader(fi1, "")
		require.NoError(err)
		h.Name = ""
		newH, err := tar.FileInfoHeader(fi2, "")
		require.NoError(err)
		newH.Name = ""
		similar, err := IsSimilarHeader(h, newH)
		require.True(similar)
		require.NoError(err)
	})

	t.Run("DirAndSymlinkConsideredDifferent", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		testDir1, err := ioutil.TempDir(tmpRoot, "test1")
		require.NoError(err)
		testLink2 := path.Join(tmpRoot, "test_link")
		require.NoError(os.Symlink(testDir1, testLink2))

		fi1, err := os.Lstat(testDir1)
		require.NoError(err)
		fi2, err := os.Lstat(testLink2)
		require.NoError(err)

		h, err := tar.FileInfoHeader(fi1, "")
		require.NoError(err)
		newH, err := tar.FileInfoHeader(fi2, "")
		require.NoError(err)
		newH.Linkname = testDir1
		similar, err := IsSimilarHeader(h, newH)
		require.False(similar)
		require.NoError(err)
	})

	t.Run("FileAndSymlinkConsideredDifferent", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		testFile1, err := ioutil.TempFile(tmpRoot, "test1")
		require.NoError(err)
		testLink2 := path.Join(tmpRoot, "test_link")
		require.NoError(os.Symlink(testFile1.Name(), testLink2))

		os.Chmod(testFile1.Name(), 0755)
		atime := time.Now()
		mtime := time.Now()
		require.NoError(os.Chtimes(testFile1.Name(), atime, mtime))
		require.NoError(os.Chtimes(testLink2, atime, mtime))

		fi1, err := os.Lstat(testFile1.Name())
		require.NoError(err)
		fi2, err := os.Lstat(testLink2)
		require.NoError(err)

		h, err := tar.FileInfoHeader(fi1, "")
		require.NoError(err)
		newH, err := tar.FileInfoHeader(fi2, "")
		require.NoError(err)
		newH.Linkname = testFile1.Name()
		similar, err := IsSimilarHeader(h, newH)
		require.False(similar)
		require.NoError(err)
	})

	t.Run("FileAndHardLinkConsideredDifferent", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		testFile1, err := ioutil.TempFile(tmpRoot, "test1")
		require.NoError(err)
		testLink2 := path.Join(tmpRoot, "test_link")
		require.NoError(os.Link(testFile1.Name(), testLink2))

		os.Chmod(testFile1.Name(), 0755)
		atime := time.Now()
		mtime := time.Now()
		require.NoError(os.Chtimes(testFile1.Name(), atime, mtime))
		require.NoError(os.Chtimes(testLink2, atime, mtime))

		fi1, err := os.Lstat(testFile1.Name())
		require.NoError(err)
		fi2, err := os.Lstat(testLink2)
		require.NoError(err)

		h, err := tar.FileInfoHeader(fi1, "")
		require.NoError(err)
		newH, err := tar.FileInfoHeader(fi2, "")
		require.NoError(err)
		newH.Typeflag = tar.TypeLink
		newH.Linkname = testFile1.Name()
		newH.Size = 0

		similar, err := IsSimilarHeader(h, newH)
		require.False(similar)
		require.NoError(err)
	})
}

func TestIsSimilarSymlink(t *testing.T) {
	t.Run("NoChange", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		testFile, err := ioutil.TempFile(tmpRoot, "test")
		require.NoError(err)
		testLink1 := path.Join(tmpRoot, "test_link1")
		require.NoError(os.Symlink(testFile.Name(), testLink1))
		testLink2 := path.Join(tmpRoot, "test_link2")
		require.NoError(os.Symlink(testFile.Name(), testLink2))

		atime := time.Now().Add(-time.Hour)
		mtime := time.Now().Add(-time.Hour * 2)
		require.NoError(os.Chtimes(testLink1, atime, mtime))
		require.NoError(os.Chtimes(testLink2, atime, mtime))

		fi1, err := os.Lstat(testLink1)
		require.NoError(err)
		fi2, err := os.Lstat(testLink2)
		require.NoError(err)

		h, err := tar.FileInfoHeader(fi1, "")
		require.NoError(err)
		h.Linkname = testFile.Name()
		newH, err := tar.FileInfoHeader(fi2, "")
		require.NoError(err)
		newH.Linkname = testFile.Name()
		similar, err := isSimilarSymlink(h, newH)
		require.True(similar)
		require.NoError(err)
		similar, err = IsSimilarHeader(h, newH)
		require.True(similar)
		require.NoError(err)
	})

	t.Run("DifferentLinkTargetConsideredDifferent", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		testFile1, err := ioutil.TempFile(tmpRoot, "test")
		require.NoError(err)
		testFile2, err := ioutil.TempFile(tmpRoot, "test2")
		require.NoError(err)
		testLink1 := path.Join(tmpRoot, "test_link1")
		require.NoError(os.Symlink(testFile1.Name(), testLink1))
		testLink2 := path.Join(tmpRoot, "test_link2")
		require.NoError(os.Symlink(testFile2.Name(), testLink2))

		atime := time.Now().Add(-time.Hour)
		mtime := time.Now().Add(-time.Hour * 2)
		require.NoError(os.Chtimes(testLink1, atime, mtime))
		require.NoError(os.Chtimes(testLink2, atime, mtime))

		fi1, err := os.Lstat(testLink1)
		require.NoError(err)
		fi2, err := os.Lstat(testLink2)
		require.NoError(err)

		h, err := tar.FileInfoHeader(fi1, "")
		require.NoError(err)
		h.Linkname = testFile1.Name()
		newH, err := tar.FileInfoHeader(fi2, "")
		require.NoError(err)
		newH.Linkname = testFile2.Name()
		similar, err := isSimilarSymlink(h, newH)
		require.False(similar)
		require.NoError(err)
		similar, err = IsSimilarHeader(h, newH)
		require.False(similar)
		require.NoError(err)
	})
}

func TestIsSimilarHardlink(t *testing.T) {
	t.Run("NoChange", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		testFile, err := ioutil.TempFile(tmpRoot, "test")
		require.NoError(err)
		testLink1 := path.Join(tmpRoot, "test_link1")
		require.NoError(os.Link(testFile.Name(), testLink1))
		testLink2 := path.Join(tmpRoot, "test_link2")
		require.NoError(os.Link(testFile.Name(), testLink2))

		atime := time.Now().Add(-time.Hour)
		mtime := time.Now().Add(-time.Hour * 2)
		require.NoError(os.Chtimes(testLink1, atime, mtime))
		require.NoError(os.Chtimes(testLink2, atime, mtime))

		fi1, err := os.Lstat(testLink1)
		require.NoError(err)
		fi2, err := os.Lstat(testLink2)
		require.NoError(err)

		h, err := tar.FileInfoHeader(fi1, "")
		require.NoError(err)
		h.Typeflag = tar.TypeLink
		h.Linkname = testFile.Name()
		h.Size = 0
		newH, err := tar.FileInfoHeader(fi2, "")
		require.NoError(err)
		newH.Typeflag = tar.TypeLink
		newH.Linkname = testFile.Name()
		newH.Size = 0
		similar, err := isSimilarHardLink(h, newH)
		require.True(similar)
		require.NoError(err)
		similar, err = IsSimilarHeader(h, newH)
		require.True(similar)
		require.NoError(err)
	})

	t.Run("DifferentLinkTargetConsideredDifferent", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		testFile1, err := ioutil.TempFile(tmpRoot, "test")
		require.NoError(err)
		testFile2, err := ioutil.TempFile(tmpRoot, "test2")
		require.NoError(err)
		testLink1 := path.Join(tmpRoot, "test_link1")
		require.NoError(os.Link(testFile1.Name(), testLink1))
		testLink2 := path.Join(tmpRoot, "test_link2")
		require.NoError(os.Link(testFile2.Name(), testLink2))

		atime := time.Now().Add(-time.Hour)
		mtime := time.Now().Add(-time.Hour * 2)
		require.NoError(os.Chtimes(testLink1, atime, mtime))
		require.NoError(os.Chtimes(testLink2, atime, mtime))

		fi1, err := os.Lstat(testLink1)
		require.NoError(err)
		fi2, err := os.Lstat(testLink2)
		require.NoError(err)

		h, err := tar.FileInfoHeader(fi1, "")
		require.NoError(err)
		h.Typeflag = tar.TypeLink
		h.Linkname = testFile1.Name()
		h.Size = 0
		newH, err := tar.FileInfoHeader(fi2, "")
		require.NoError(err)
		newH.Typeflag = tar.TypeLink
		newH.Linkname = testFile2.Name()
		newH.Size = 0
		similar, err := isSimilarHardLink(h, newH)
		require.False(similar)
		require.NoError(err)
		similar, err = IsSimilarHeader(h, newH)
		require.False(similar)
		require.NoError(err)
	})
}

func TestIsSimilarDirectory(t *testing.T) {
	t.Run("NoChange", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		testDir1, err := ioutil.TempDir(tmpRoot, "test1")
		require.NoError(err)
		testDir2, err := ioutil.TempDir(tmpRoot, "test2")
		require.NoError(err)

		fi1, err := os.Lstat(testDir1)
		require.NoError(err)
		fi2, err := os.Lstat(testDir2)
		require.NoError(err)

		h, err := tar.FileInfoHeader(fi1, "")
		require.NoError(err)
		newH, err := tar.FileInfoHeader(fi2, "")
		require.NoError(err)
		similar, err := isSimilarDirectory(h, newH)
		require.True(similar)
		require.NoError(err)
		similar, err = IsSimilarHeader(h, newH)
		require.True(similar)
		require.NoError(err)
	})

	t.Run("DifferentContentConsideredSimilar", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		testDir1, err := ioutil.TempDir(tmpRoot, "test1")
		require.NoError(err)
		testDir2, err := ioutil.TempDir(tmpRoot, "test2")
		require.NoError(err)

		_, err = ioutil.TempFile(testDir2, "test")
		require.NoError(err)

		fi1, err := os.Lstat(testDir1)
		require.NoError(err)
		fi2, err := os.Lstat(testDir2)
		require.NoError(err)

		h, err := tar.FileInfoHeader(fi1, "")
		require.NoError(err)
		newH, err := tar.FileInfoHeader(fi2, "")
		require.NoError(err)
		similar, err := isSimilarDirectory(h, newH)
		require.True(similar)
		require.NoError(err)
		similar, err = IsSimilarHeader(h, newH)
		require.True(similar)
		require.NoError(err)
	})

	t.Run("DifferentModTimeConsideredDifferent", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		testDir1, err := ioutil.TempDir(tmpRoot, "test1")
		require.NoError(err)
		testDir2, err := ioutil.TempDir(tmpRoot, "test2")
		require.NoError(err)

		atime := time.Now().Add(-time.Hour)
		mtime := time.Now().Add(-time.Hour * 2)
		require.NoError(os.Chtimes(testDir1, atime, mtime))

		fi1, err := os.Lstat(testDir1)
		require.NoError(err)
		fi2, err := os.Lstat(testDir2)
		require.NoError(err)

		h, err := tar.FileInfoHeader(fi1, "")
		require.NoError(err)
		newH, err := tar.FileInfoHeader(fi2, "")
		require.NoError(err)
		similar, err := isSimilarDirectory(h, newH)
		require.False(similar)
		require.NoError(err)
		similar, err = IsSimilarHeader(h, newH)
		require.False(similar)
		require.NoError(err)
	})

	t.Run("DifferentModeConsideredDifferent", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		testDir1, err := ioutil.TempDir(tmpRoot, "test1")
		require.NoError(err)
		testDir2, err := ioutil.TempDir(tmpRoot, "test2")
		require.NoError(err)

		require.NoError(os.Chmod(testDir1, os.FileMode(0777)))

		fi1, err := os.Lstat(testDir1)
		require.NoError(err)
		fi2, err := os.Lstat(testDir2)
		require.NoError(err)

		h, err := tar.FileInfoHeader(fi1, "")
		require.NoError(err)
		newH, err := tar.FileInfoHeader(fi2, "")
		require.NoError(err)
		similar, err := isSimilarDirectory(h, newH)
		require.False(similar)
		require.NoError(err)
		similar, err = IsSimilarHeader(h, newH)
		require.False(similar)
		require.NoError(err)
	})
}

func TestIsSimilarRegularFile(t *testing.T) {
	t.Run("NoChange", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		testFile1, err := ioutil.TempFile(tmpRoot, "test1")
		require.NoError(err)
		testFile2, err := ioutil.TempFile(tmpRoot, "test2")
		require.NoError(err)

		atime := time.Now().Add(-time.Hour)
		mtime := time.Now().Add(-time.Hour * 2)
		require.NoError(os.Chtimes(testFile1.Name(), atime, mtime))
		require.NoError(os.Chtimes(testFile2.Name(), atime, mtime))

		fi1, err := os.Lstat(testFile1.Name())
		require.NoError(err)
		fi2, err := os.Lstat(testFile2.Name())
		require.NoError(err)

		h, err := tar.FileInfoHeader(fi1, "")
		require.NoError(err)
		newH, err := tar.FileInfoHeader(fi2, "")
		require.NoError(err)
		similar, err := isSimilarRegularFile(h, newH)
		require.True(similar)
		require.NoError(err)
		similar, err = IsSimilarHeader(h, newH)
		require.True(similar)
		require.NoError(err)
	})

	t.Run("DifferentContentButSameSizeConsideredSimilar", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		testFile1, err := ioutil.TempFile(tmpRoot, "test1")
		require.NoError(err)
		testFile2, err := ioutil.TempFile(tmpRoot, "test2")
		require.NoError(err)

		testFile1.Write([]byte("test1"))
		testFile1.Close()
		testFile2.Write([]byte("test2"))
		testFile2.Close()
		atime := time.Now().Add(-time.Hour)
		mtime := time.Now().Add(-time.Hour * 2)
		require.NoError(os.Chtimes(testFile1.Name(), atime, mtime))
		require.NoError(os.Chtimes(testFile2.Name(), atime, mtime))

		fi1, err := os.Lstat(testFile1.Name())
		require.NoError(err)
		fi2, err := os.Lstat(testFile2.Name())
		require.NoError(err)

		h, err := tar.FileInfoHeader(fi1, "")
		require.NoError(err)
		newH, err := tar.FileInfoHeader(fi2, "")
		require.NoError(err)
		similar, err := isSimilarRegularFile(h, newH)
		require.True(similar)
		require.NoError(err)
		similar, err = IsSimilarHeader(h, newH)
		require.True(similar)
		require.NoError(err)
	})

	t.Run("DifferentSizeConsideredDifferent", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		testFile1, err := ioutil.TempFile(tmpRoot, "test1")
		require.NoError(err)
		testFile2, err := ioutil.TempFile(tmpRoot, "test2")
		require.NoError(err)

		os.Truncate(testFile2.Name(), 100)
		atime := time.Now().Add(-time.Hour)
		mtime := time.Now().Add(-time.Hour * 2)
		require.NoError(os.Chtimes(testFile1.Name(), atime, mtime))
		require.NoError(os.Chtimes(testFile2.Name(), atime, mtime))

		fi1, err := os.Lstat(testFile1.Name())
		require.NoError(err)
		fi2, err := os.Lstat(testFile2.Name())
		require.NoError(err)

		h, err := tar.FileInfoHeader(fi1, "")
		require.NoError(err)
		newH, err := tar.FileInfoHeader(fi2, "")
		require.NoError(err)
		similar, err := isSimilarRegularFile(h, newH)
		require.False(similar)
		require.NoError(err)
		similar, err = IsSimilarHeader(h, newH)
		require.False(similar)
		require.NoError(err)
	})

	t.Run("DifferentModTimeConsideredDifferent", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		testFile1, err := ioutil.TempFile(tmpRoot, "test1")
		require.NoError(err)
		testFile2, err := ioutil.TempFile(tmpRoot, "test2")
		require.NoError(err)

		atime := time.Now().Add(-time.Hour)
		mtime := time.Now().Add(-time.Hour * 2)
		require.NoError(os.Chtimes(testFile1.Name(), atime, mtime))

		fi1, err := os.Lstat(testFile1.Name())
		require.NoError(err)
		fi2, err := os.Lstat(testFile2.Name())
		require.NoError(err)

		h, err := tar.FileInfoHeader(fi1, "")
		require.NoError(err)
		newH, err := tar.FileInfoHeader(fi2, "")
		require.NoError(err)
		similar, err := isSimilarRegularFile(h, newH)
		require.False(similar)
		require.NoError(err)
		similar, err = IsSimilarHeader(h, newH)
		require.False(similar)
		require.NoError(err)
	})

	t.Run("DifferentModeConsideredDifferent", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		testFile1, err := ioutil.TempFile(tmpRoot, "test1")
		require.NoError(err)
		testFile2, err := ioutil.TempFile(tmpRoot, "test2")
		require.NoError(err)

		require.NoError(os.Chmod(testFile1.Name(), os.FileMode(0777)))

		fi1, err := os.Lstat(testFile1.Name())
		require.NoError(err)
		fi2, err := os.Lstat(testFile2.Name())
		require.NoError(err)

		h, err := tar.FileInfoHeader(fi1, "")
		require.NoError(err)
		newH, err := tar.FileInfoHeader(fi2, "")
		require.NoError(err)
		similar, err := isSimilarRegularFile(h, newH)
		require.False(similar)
		require.NoError(err)
		similar, err = IsSimilarHeader(h, newH)
		require.False(similar)
		require.NoError(err)
	})
}
