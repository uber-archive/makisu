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
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/uber/makisu/lib/pathutils"

	"github.com/andres-erbsen/clock"
	"github.com/stretchr/testify/require"
)

func TestUntarFromPath(t *testing.T) {
	require := require.New(t)

	tmpBase, err := ioutil.TempDir("/tmp", "makisu-test")
	require.NoError(err)
	defer os.RemoveAll(tmpBase)

	tmpRoot, err := ioutil.TempDir(tmpBase, "root")
	require.NoError(err)
	src, err := ioutil.TempDir(tmpBase, "src")
	require.NoError(err)
	src2, err := ioutil.TempDir(tmpBase, "src2")
	require.NoError(err)

	// Files in the archive.
	err = ioutil.WriteFile(filepath.Join(src, "test.txt"), []byte("TEST"), 0677) // 1
	require.NoError(err)
	err = os.Mkdir(filepath.Join(src, "test1"), os.ModePerm) // 2
	require.NoError(err)
	err = os.Mkdir(filepath.Join(src, "test2"), os.ModePerm) // 3
	require.NoError(err)
	err = ioutil.WriteFile(filepath.Join(src, "test1", "test1.txt"), []byte("TEST1"), 0677) // 4
	require.NoError(err)
	err = os.Link(filepath.Join(src, "test1", "test1.txt"), filepath.Join(src, "test2.txt")) // 5
	require.NoError(err)
	err = ioutil.WriteFile(filepath.Join(src, "target.txt"), []byte("TARGET"), 0677) // 6
	require.NoError(err)
	err = os.Symlink(filepath.Join(src, "target.txt"), filepath.Join(src, "mydir")) // 7
	require.NoError(err)

	err = CreateTarFromDirectory(filepath.Join(tmpBase, "archive1.tar"), src)
	require.NoError(err)

	// Files already existing under the memfs root.
	err = os.Mkdir(filepath.Join(tmpRoot, "test1"), os.ModePerm)
	require.NoError(err)
	err = ioutil.WriteFile(filepath.Join(tmpRoot, "test1", "test1.txt"), []byte("TEST1"), 0677)
	require.NoError(err)
	err = os.Mkdir(filepath.Join(tmpRoot, "mydir"), os.ModePerm)
	require.NoError(err)

	clk := clock.NewMock()
	fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
	require.NoError(err)
	fs.blacklist = nil

	err = fs.UpdateFromTarPath(filepath.Join(tmpBase, "archive1.tar"), true)
	require.NoError(err)

	contents, err := ioutil.ReadFile(filepath.Join(tmpRoot, "test.txt"))
	require.NoError(err)
	require.Equal([]byte("TEST"), contents)

	contents, err = ioutil.ReadFile(filepath.Join(tmpRoot, "test1", "test1.txt"))
	require.NoError(err)
	require.Equal([]byte("TEST1"), contents)

	fi, err := os.Lstat(filepath.Join(tmpRoot, "mydir"))
	require.NoError(err)
	require.False(fi.IsDir())

	contents, err = ioutil.ReadFile(filepath.Join(tmpRoot, "mydir"))
	require.NoError(err)
	require.Equal([]byte("TARGET"), contents)

	require.Equal(7, fs.layers[len(fs.layers)-1].count())

	// Whiteout files already existing in the memfs.
	err = os.Mkdir(filepath.Join(src2, ".wh.test.txt"), os.ModePerm)
	require.NoError(err)
	err = os.Mkdir(filepath.Join(src2, ".wh.test1"), os.ModePerm)
	require.NoError(err)

	err = CreateTarFromDirectory(filepath.Join(tmpBase, "archive2.tar"), src2)
	require.NoError(err)

	err = fs.UpdateFromTarPath(filepath.Join(tmpBase, "archive2.tar"), true)
	require.NoError(err)

	_, err = os.Stat(filepath.Join(tmpRoot, "test.txt"))
	require.True(os.IsNotExist(err))

	_, err = os.Stat(filepath.Join(tmpRoot, "test.txt"))
	require.True(os.IsNotExist(err))

	require.Equal(2, fs.layers[len(fs.layers)-1].count())
}

func TestMemNodeIsOnDisk(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		l := newMemLayer()
		src, err := ioutil.TempFile(tmpRoot, "test")
		require.NoError(err)
		fi, err := os.Lstat(src.Name())
		require.NoError(err)
		hdr, err := l.createHeader(tmpRoot, src.Name(), "/tmp/testDest", fi)
		require.NoError(err)

		n := newMemFSNode(newContentMemFile(src.Name(), filepath.Base(src.Name()), hdr))
		ok, err := n.isOnDisk()
		require.NoError(err)
		require.True(ok)
	})

	t.Run("Nonexistent", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		l := newMemLayer()
		src, err := ioutil.TempFile(tmpRoot, "test")
		require.NoError(err)
		fi, err := os.Lstat(src.Name())
		require.NoError(err)
		hdr, err := l.createHeader(tmpRoot, src.Name(), "/tmp/testDest", fi)
		require.NoError(err)
		os.Remove(src.Name())

		n := newMemFSNode(newContentMemFile(filepath.Base(src.Name()), src.Name(), hdr))
		ok, err := n.isOnDisk()
		require.NoError(err)
		require.False(ok)
	})
}

func TestUpdateMemFS(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		clk := clock.NewMock()
		fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
		require.NoError(err)

		l1 := newMemLayer()
		dst1 := "/test1"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst1, 0755))
		require.NoError(fs.merge(l1))

		l2 := newMemLayer()
		dst2 := "/test1/test2"
		require.NoError(addDirectoryToLayer(l2, tmpRoot, dst2, 0755))
		require.NoError(fs.merge(l2))

		n, err := findNode(fs, dst2, false, 0)
		require.NoError(err)
		require.NotNil(n)
		require.Equal(dst2, n.dst)
	})

	t.Run("Mutation", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		clk := clock.NewMock()
		fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
		require.NoError(err)

		l1 := newMemLayer()
		dst1 := "/test1"
		require.NoError(addRegularFileToLayer(l1, tmpRoot, dst1, "hello", 0755))
		require.NoError(fs.merge(l1))

		l2 := newMemLayer()
		require.NoError(addRegularFileToLayer(l2, tmpRoot, dst1, "hello", 0777))
		require.NoError(fs.merge(l2))

		n, err := findNode(fs, dst1, false, 0)
		require.NoError(err)
		require.NotNil(n)
		require.Equal(dst1, n.dst)
		require.Equal(os.FileMode(0777), os.FileMode(n.hdr.Mode).Perm())
	})

	t.Run("TrailingSlashes", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		clk := clock.NewMock()
		fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
		require.NoError(err)

		l1 := newMemLayer()
		dst1 := "test1/"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst1, 0755))
		require.NoError(fs.merge(l1))

		l2 := newMemLayer()
		dst2 := "test1/test2/"
		require.NoError(addDirectoryToLayer(l2, tmpRoot, dst2, 0755))
		require.NoError(fs.merge(l2))

		n, err := findNode(fs, "/test1/test2/", false, 0)
		require.NoError(err)
		require.NotNil(n)
		require.Equal("/test1/test2", n.dst)
	})

	t.Run("SkipDirCausesError", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		clk := clock.NewMock()
		fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
		require.NoError(err)

		l1 := newMemLayer()
		dst1 := "/test1"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst1, 0755))
		dst2 := "/test1/test2/test3"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst2, 0755))
		require.Error(fs.merge(l1))
	})

	t.Run("WhiteoutExistingDir", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		clk := clock.NewMock()
		fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
		require.NoError(err)

		l1 := newMemLayer()
		dst11 := "/test11"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst11, 0755))
		dst12 := "/test11/test12"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst12, 0755))
		dst13 := "/test11/test12/test.txt"
		require.NoError(addRegularFileToLayer(l1, tmpRoot, dst13, "hello", 0755))
		require.NoError(fs.merge(l1))

		n, err := findNode(fs, dst13, false, 0)
		require.NoError(err)
		require.NotNil(n)
		require.Equal(dst13, n.dst)

		l2 := newMemLayer()
		dst21 := "/test11"
		require.NoError(addDirectoryToLayer(l2, tmpRoot, dst21, 0755))
		dst22 := "/test11/.wh.test12"
		require.NoError(addDirectoryToLayer(l2, tmpRoot, dst22, 0755))
		require.NoError(fs.merge(l2))

		n, err = findNode(fs, dst11, false, 0)
		require.NoError(err)
		require.NotNil(n)
		require.Equal(dst11, n.dst)

		n, err = findNode(fs, dst13, false, 0)
		require.Equal(os.ErrNotExist, err)
	})

	t.Run("WhiteoutNonexistentCausesError", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		clk := clock.NewMock()
		fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
		require.NoError(err)

		l1 := newMemLayer()
		dst11 := "/test11"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst11, 0755))
		dst12 := "/test11/test12"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst12, 0755))
		dst13 := "/test11/test12/test.txt"
		require.NoError(addRegularFileToLayer(l1, tmpRoot, dst13, "hello", 0755))
		require.NoError(fs.merge(l1))

		n, err := findNode(fs, dst13, false, 0)
		require.NoError(err)
		require.NotNil(n)
		require.Equal(dst13, n.dst)

		l2 := newMemLayer()
		dst21 := "/test11"
		require.NoError(addDirectoryToLayer(l2, tmpRoot, dst21, 0755))
		dst22 := "/test11/.wh.test13"
		require.NoError(addDirectoryToLayer(l2, tmpRoot, dst22, 0755))
		require.Error(fs.merge(l2))
	})
}

func TestGetAncestors(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		clk := clock.NewMock()
		fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
		require.NoError(err)

		l1 := newMemLayer()
		dst1 := "/test1"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst1, 0755))
		require.NoError(fs.merge(l1))

		l2 := newMemLayer()
		dst2 := "/test1/test2"
		require.NoError(addDirectoryToLayer(l2, tmpRoot, dst2, 0755))
		require.NoError(fs.merge(l2))

		n1, err := findNode(fs, dst1, false, 0)
		require.NoError(err)

		resolved, err := fs.addAncestors(l2, dst2, false, 0, 0, 0)
		require.Equal(dst2, resolved)
		require.NoError(err)
		require.Len(l2.files, 2)
		require.Contains(l2.files, n1.dst)
	})

	t.Run("Inclusive", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		clk := clock.NewMock()
		fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
		require.NoError(err)

		l1 := newMemLayer()
		dst1 := "/test1"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst1, 0766))
		dst2 := "/test1/test2"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst2, 0777))
		require.NoError(fs.merge(l1))

		n1, err := findNode(fs, dst1, false, 0)
		require.NoError(err)
		n2, err := findNode(fs, dst2, false, 0)
		require.NoError(err)

		resolved, err := fs.addAncestors(l1, dst2, true, 0, 0, 0)
		require.Equal(dst2, resolved)
		require.NoError(err)
		require.Len(l1.files, 2)
		require.Contains(l1.files, n1.dst)
		require.Contains(l1.files, n2.dst)
		require.Equal(os.FileMode(0766), os.FileMode(n1.hdr.Mode).Perm())
		require.Equal(os.FileMode(0777), os.FileMode(n2.hdr.Mode).Perm())
	})

	t.Run("FillNonexistent", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		clk := clock.NewMock()
		fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
		require.NoError(err)

		l1 := newMemLayer()
		dst1 := "/test1"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst1, 0755))
		require.NoError(fs.merge(l1))

		l2 := newMemLayer()
		dst2 := "/test1/test2"
		require.NoError(addDirectoryToLayer(l2, tmpRoot, dst2, 0755))
		require.NoError(fs.merge(l2))

		resolved, err := fs.addAncestors(l2, "/nonexistent1/nonexistent2", false, 0, 0, 0)
		require.Equal("/nonexistent1/nonexistent2", resolved)
		require.NoError(err)

		n1, err := findNode(fs, "/nonexistent1", false, 0)
		require.NoError(err)

		require.Len(l2.files, 2)
		require.Contains(l2.files, n1.dst)
	})

	t.Run("FollowSymlinkFullResolve", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		clk := clock.NewMock()
		fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
		require.NoError(err)

		l1 := newMemLayer()
		dst11 := "/test11"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst11, 0755))
		dst12 := "/test11/test12"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst12, 0755))
		dst13 := "/test11/test12/ignore1"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst13, 0755))
		require.NoError(fs.merge(l1))

		l2 := newMemLayer()
		dst21 := "/test21"
		require.NoError(addDirectoryToLayer(l2, tmpRoot, dst21, 0755))
		dst22 := "/test21/test22"
		require.NoError(addDirectoryToLayer(l2, tmpRoot, dst22, 0755))
		dst23 := "/test21/test22/link"
		require.NoError(addSymlinkToLayer(l2, tmpRoot, dst23, dst11))
		dst24 := "/test21/test22/ignore2"
		require.NoError(addDirectoryToLayer(l2, tmpRoot, dst24, 0755))
		require.NoError(fs.merge(l2))

		n11, err := findNode(fs, dst11, false, 0)
		require.NoError(err)
		n21, err := findNode(fs, dst21, false, 0)
		require.NoError(err)
		n22, err := findNode(fs, dst22, false, 0)
		require.NoError(err)
		n23, err := findNode(fs, dst23, false, 0)
		require.NoError(err)

		resolved, err := fs.addAncestors(l2, "/test21/test22/link/test12", false, 0, 0, 0)
		require.Equal("/test11/test12", resolved)
		require.NoError(err)
		require.Len(l2.files, 5)
		require.Contains(l2.files, n21.contentMemFile.dst)
		require.Contains(l2.files, n22.contentMemFile.dst)
		require.Contains(l2.files, n23.contentMemFile.dst)
		require.Contains(l2.files, n11.contentMemFile.dst)
	})

	t.Run("FollowSymlinkPartialResolve", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		clk := clock.NewMock()
		fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
		require.NoError(err)

		l1 := newMemLayer()
		dst11 := "/test11"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst11, 0755))
		dst12 := "/test11/test12"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst12, 0777))
		require.NoError(fs.merge(l1))

		l2 := newMemLayer()
		dst21 := "/test21"
		require.NoError(addDirectoryToLayer(l2, tmpRoot, dst21, 0755))
		dst22 := "/test21/test22"
		require.NoError(addDirectoryToLayer(l2, tmpRoot, dst22, 0755))
		dst23 := "/test21/test22/link"
		require.NoError(addSymlinkToLayer(l2, tmpRoot, dst23, dst11))
		require.NoError(fs.merge(l2))

		n11, err := findNode(fs, dst11, false, 0)
		require.NoError(err)
		n12, err := findNode(fs, dst12, false, 0)
		require.NoError(err)
		n21, err := findNode(fs, dst21, false, 0)
		require.NoError(err)
		n22, err := findNode(fs, dst22, false, 0)
		require.NoError(err)
		n23, err := findNode(fs, dst23, false, 0)
		require.NoError(err)

		resolved, err := fs.addAncestors(l2, "/test21/test22/link/test12/test13/nonexistent", false, 0, 0, 0)
		require.Equal("/test11/test12/test13/nonexistent", resolved)
		require.NoError(err)
		require.Len(l2.files, 6)

		hdr13, err := l2.createHeader(tmpRoot, "", "/test11/test12/test13", n12.hdr.FileInfo())
		hdr13.ModTime = clk.Now()
		require.NoError(err)

		require.Contains(l2.files, n21.contentMemFile.dst)
		require.Contains(l2.files, n22.contentMemFile.dst)
		require.Contains(l2.files, n23.contentMemFile.dst)
		require.Contains(l2.files, n11.contentMemFile.dst)
		require.Contains(l2.files, n12.contentMemFile.dst)
		require.Contains(l2.files, newContentMemFile("", "/test11/test12/test13", hdr13).dst)
	})

	t.Run("DetectSymlinkInfiniteLoop", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		clk := clock.NewMock()
		fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
		require.NoError(err)

		l1 := newMemLayer()
		dst11 := "/test11"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst11, 0755))
		dst12 := "/test11/link"
		require.NoError(addSymlinkToLayer(l1, tmpRoot, dst12, "/test21/link"))
		require.NoError(fs.merge(l1))

		l2 := newMemLayer()
		dst21 := "/test21"
		require.NoError(addDirectoryToLayer(l2, tmpRoot, dst21, 0755))
		dst22 := "/test21/link"
		require.NoError(addSymlinkToLayer(l2, tmpRoot, dst22, "/test11/link"))
		require.NoError(fs.merge(l2))

		_, err = fs.addAncestors(l2, "/test21/link/nonexistent", false, 0, 0, 0)
		require.Error(err)
	})
}

func TestCreateLayerByScan(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		clk := clock.NewMock()
		fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
		require.NoError(err)
		fs.blacklist = nil

		l1 := newMemLayer()
		dst11 := "/test1"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst11, 0755))
		dst12 := "/test1/test.txt"
		require.NoError(addRegularFileToLayer(l1, tmpRoot, dst12, "hello", 0755))
		l, err := fs.createLayerByScan()
		require.NoError(err)
		requireEqualLayers(require, l1, l)

		l2 := newMemLayer()
		require.NoError(addDirectoryToLayer(l2, tmpRoot, dst11, 0755))
		dst22 := "/test1/test2"
		require.NoError(addDirectoryToLayer(l2, tmpRoot, dst22, 0755))
		dst23 := "/test1/test2/test3"
		require.NoError(addDirectoryToLayer(l2, tmpRoot, dst23, 0755))
		l, err = fs.createLayerByScan()
		require.NoError(err)
		requireEqualLayers(require, l2, l)
	})

	t.Run("Symlink", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		clk := clock.NewMock()
		fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
		require.NoError(err)
		fs.blacklist = nil

		l1 := newMemLayer()
		dst11 := "/test11"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst11, 0755))
		dst12 := "/test11/test12"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst12, 0755))
		dst13 := "/test11/test12/ignore1"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst13, 0755))
		l, err := fs.createLayerByScan()
		require.NoError(err)
		requireEqualLayers(require, l1, l)

		l2 := newMemLayer()
		dst21 := "/test21"
		require.NoError(addDirectoryToLayer(l2, tmpRoot, dst21, 0755))
		dst22 := "/test21/test22"
		require.NoError(addDirectoryToLayer(l2, tmpRoot, dst22, 0755))
		dst23 := "/test21/test22/link"
		require.NoError(addSymlinkToLayer(l2, tmpRoot, dst23, dst11))
		dst24 := "/test21/test22/ignore2"
		require.NoError(addDirectoryToLayer(l2, tmpRoot, dst24, 0755))
		l, err = fs.createLayerByScan()
		require.NoError(err)
		requireEqualLayers(require, l2, l)
	})

	t.Run("Whiteout", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		clk := clock.NewMock()
		fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
		require.NoError(err)
		fs.blacklist = nil

		l1 := newMemLayer()
		dst11 := "/test11"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst11, 0755))
		dst12 := "/test11/test12"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst12, 0755))
		dst13 := "/test11/test12/test.txt"
		require.NoError(addRegularFileToLayer(l1, tmpRoot, dst13, "hello", 0755))
		dst14 := "/test11/test14.txt"
		require.NoError(addRegularFileToLayer(l1, tmpRoot, dst14, "hello", 0755))
		l, err := fs.createLayerByScan()
		require.NoError(err)
		requireEqualLayers(require, l1, l)

		n, err := findNode(fs, dst13, false, 0)
		require.NoError(err)
		require.NotNil(n)
		require.Equal(dst13, n.dst)

		l2 := newMemLayer()
		require.NoError(addDirectoryToLayer(l2, tmpRoot, dst11, 0755))
		dst22 := "/test11/.wh.test12"
		os.RemoveAll(filepath.Join(tmpRoot, dst12))
		require.NoError(addDirectoryToLayer(l2, tmpRoot, dst22, 0755))
		os.RemoveAll(filepath.Join(tmpRoot, dst22))
		dst24 := "/test11/.wh.test14.txt"
		os.RemoveAll(filepath.Join(tmpRoot, dst14))
		require.NoError(addDirectoryToLayer(l2, tmpRoot, dst24, 0755))
		os.RemoveAll(filepath.Join(tmpRoot, dst24))
		l, err = fs.createLayerByScan()
		require.NoError(err)
		requireEqualLayers(require, l2, l)
	})
}

func TestCreateLayerByCopy(t *testing.T) {
	t.Run("file dir/file", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		clk := clock.NewMock()
		fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
		require.NoError(err)
		fs.blacklist = nil

		l1 := newMemLayer()
		dst11 := "/test1"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst11, 0755))
		dst12 := "/test1/test.txt"
		require.NoError(addRegularFileToLayer(l1, tmpRoot, dst12, "hello", 0755))
		require.NoError(fs.merge(l1))

		srcs := []string{"/test1/test.txt"}
		srcRoot := tmpRoot
		workDir := ""
		dst := "/test2/test.txt"
		c, err := NewCopyOperation(
			srcs, srcRoot, workDir, dst, validChown, pathutils.DefaultBlacklist, false)
		require.NoError(err)
		err = fs.addToLayer(newMemLayer(), c)
		require.NoError(err)

		n1, err := findNode(fs, "/test2", false, 0)
		require.NoError(err)
		require.NotNil(n1)
		require.Equal("/test2", n1.dst)

		n2, err := findNode(fs, dst, false, 0)
		require.NoError(err)
		require.NotNil(n2)
		require.Equal(tmpRoot+"/test1/test.txt", n2.src)
	})

	t.Run("file dir/", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		clk := clock.NewMock()
		fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
		require.NoError(err)
		fs.blacklist = nil

		l1 := newMemLayer()
		dst11 := "/test1"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst11, 0755))
		dst12 := "/test1/test.txt"
		require.NoError(addRegularFileToLayer(l1, tmpRoot, dst12, "hello", 0755))
		require.NoError(fs.merge(l1))

		srcs := []string{"/test1/test.txt"}
		srcRoot := tmpRoot
		workDir := ""
		dst := "/dst/"
		c, err := NewCopyOperation(
			srcs, srcRoot, workDir, dst, validChown, pathutils.DefaultBlacklist, false)
		require.NoError(err)
		err = fs.addToLayer(newMemLayer(), c)
		require.NoError(err)

		n1, err := findNode(fs, "/dst", false, 0)
		require.NoError(err)
		require.NotNil(n1)
		require.Equal("/dst", n1.dst)

		n2, err := findNode(fs, "/dst/test.txt", false, 0)
		require.NoError(err)
		require.NotNil(n2)
		require.Equal(tmpRoot+"/test1/test.txt", n2.src)
	})

	t.Run("file file dir/", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		clk := clock.NewMock()
		fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
		require.NoError(err)
		fs.blacklist = nil

		l1 := newMemLayer()
		dst11 := "/test1"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst11, 0755))
		dst12 := "/test1/test2.txt"
		require.NoError(addRegularFileToLayer(l1, tmpRoot, dst12, "hello", 0755))
		dst13 := "/test1/test3.txt"
		require.NoError(addRegularFileToLayer(l1, tmpRoot, dst13, "hello", 0755))
		require.NoError(fs.merge(l1))

		srcs := []string{"/test1/test2.txt", "/test1/test3.txt"}
		srcRoot := tmpRoot
		workDir := ""
		dst := "/dst/"
		c, err := NewCopyOperation(
			srcs, srcRoot, workDir, dst, validChown, pathutils.DefaultBlacklist, false)
		require.NoError(err)
		err = fs.addToLayer(newMemLayer(), c)
		require.NoError(err)

		n1, err := findNode(fs, "/dst", false, 0)
		require.NoError(err)
		require.NotNil(n1)
		require.Equal("/dst", n1.dst)

		n2, err := findNode(fs, "/dst/test2.txt", false, 0)
		require.NoError(err)
		require.NotNil(n2)
		require.Equal(tmpRoot+"/test1/test2.txt", n2.src)

		n3, err := findNode(fs, "/dst/test3.txt", false, 0)
		require.NoError(err)
		require.NotNil(n3)
		require.Equal(tmpRoot+"/test1/test3.txt", n3.src)
	})

	t.Run("dir dir/", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		clk := clock.NewMock()
		fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
		require.NoError(err)
		fs.blacklist = nil

		l1 := newMemLayer()
		dst11 := "/test1"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst11, 0755))
		dst12 := "/test1/test2"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst12, 0755))
		dst13 := "/test1/test2/test.txt"
		require.NoError(addRegularFileToLayer(l1, tmpRoot, dst13, "hello", 0755))
		require.NoError(fs.merge(l1))

		srcs := []string{"/test1/test2"}
		srcRoot := tmpRoot
		workDir := ""
		dst := "/dst/"
		c, err := NewCopyOperation(
			srcs, srcRoot, workDir, dst, validChown, pathutils.DefaultBlacklist, false)
		require.NoError(err)
		err = fs.addToLayer(newMemLayer(), c)
		require.NoError(err)

		n, err := findNode(fs, "/dst", false, 0)
		require.NoError(err)
		require.NotNil(n)
		require.Equal("/dst", n.dst)

		n, err = findNode(fs, "/dst/test.txt", false, 0)
		require.NoError(err)
		require.NotNil(n)
		require.Equal(tmpRoot+"/test1/test2/test.txt", n.src)
	})

	t.Run("dir dir dir/", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		clk := clock.NewMock()
		fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
		require.NoError(err)
		fs.blacklist = nil

		l1 := newMemLayer()
		dst11 := "/test1"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst11, 0755))
		dst12 := "/test1/test2"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst12, 0755))
		dst13 := "/test1/test2/test3.txt"
		require.NoError(addRegularFileToLayer(l1, tmpRoot, dst13, "hello", 0755))
		dst14 := "/test1/test4"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst14, 0755))
		dst15 := "/test1/test4/test5"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst15, 0755))
		dst16 := "/test1/test4/test5/test6.txt"
		require.NoError(addRegularFileToLayer(l1, tmpRoot, dst16, "hello", 0755))
		require.NoError(fs.merge(l1))

		srcs := []string{"/test1/test2", "/test1/test4"}
		srcRoot := tmpRoot
		workDir := ""
		dst := "/dst/"
		c, err := NewCopyOperation(
			srcs, srcRoot, workDir, dst, validChown, pathutils.DefaultBlacklist, false)
		require.NoError(err)
		err = fs.addToLayer(newMemLayer(), c)
		require.NoError(err)

		n, err := findNode(fs, "/dst", false, 0)
		require.NoError(err)
		require.NotNil(n)
		require.Equal("/dst", n.dst)

		n, err = findNode(fs, "/dst/test3.txt", false, 0)
		require.NoError(err)
		require.NotNil(n)
		require.Equal(tmpRoot+"/test1/test2/test3.txt", n.src)

		n, err = findNode(fs, "/dst/test5", false, 0)
		require.NoError(err)
		require.NotNil(n)
		require.Equal(tmpRoot+"/test1/test4/test5", n.src)

		n, err = findNode(fs, "/dst/test5/test6.txt", false, 0)
		require.NoError(err)
		require.NotNil(n)
		require.Equal(tmpRoot+"/test1/test4/test5/test6.txt", n.src)
	})

	t.Run("file dir dir/", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		clk := clock.NewMock()
		fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
		require.NoError(err)
		fs.blacklist = nil

		l1 := newMemLayer()
		dst11 := "/test1"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst11, 0755))
		dst12 := "/test1/test2"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst12, 0755))
		dst13 := "/test1/test2/test3.txt"
		require.NoError(addRegularFileToLayer(l1, tmpRoot, dst13, "hello", 0755))
		dst14 := "/test1/test4"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst14, 0755))
		dst15 := "/test1/test4/test5"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst15, 0755))
		dst16 := "/test1/test4/test5/test6.txt"
		require.NoError(addRegularFileToLayer(l1, tmpRoot, dst16, "hello", 0755))
		require.NoError(fs.merge(l1))

		srcs := []string{"/test1/test2/test3.txt", "/test1/test4"}
		srcRoot := tmpRoot
		workDir := ""
		dst := "/dst/"
		c, err := NewCopyOperation(
			srcs, srcRoot, workDir, dst, validChown, pathutils.DefaultBlacklist, false)
		require.NoError(err)
		err = fs.addToLayer(newMemLayer(), c)
		require.NoError(err)

		n, err := findNode(fs, "/dst", false, 0)
		require.NoError(err)
		require.NotNil(n)
		require.Equal("/dst", n.dst)

		n, err = findNode(fs, "/dst/test3.txt", false, 0)
		require.NoError(err)
		require.NotNil(n)
		require.Equal(tmpRoot+"/test1/test2/test3.txt", n.src)

		n, err = findNode(fs, "/dst/test5", false, 0)
		require.NoError(err)
		require.NotNil(n)
		require.Equal(tmpRoot+"/test1/test4/test5", n.src)

		n, err = findNode(fs, "/dst/test5/test6.txt", false, 0)
		require.NoError(err)
		require.NotNil(n)
		require.Equal(tmpRoot+"/test1/test4/test5/test6.txt", n.src)
	})

	t.Run("workdir", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		clk := clock.NewMock()
		fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
		require.NoError(err)
		fs.blacklist = nil

		l1 := newMemLayer()
		dst11 := "/test1"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst11, 0755))
		dst12 := "/test1/test2"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst12, 0755))
		dst13 := "/test1/test2/test3.txt"
		require.NoError(addRegularFileToLayer(l1, tmpRoot, dst13, "hello", 0755))
		dst14 := "/test1/test4"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst14, 0755))
		dst15 := "/test1/test4/test5"
		require.NoError(addDirectoryToLayer(l1, tmpRoot, dst15, 0755))
		dst16 := "/test1/test4/test5/test6.txt"
		require.NoError(addRegularFileToLayer(l1, tmpRoot, dst16, "hello", 0755))
		require.NoError(fs.merge(l1))

		srcs := []string{"/test1/test2/test3.txt", "/test1/test4"}
		srcRoot := tmpRoot
		workDir := "/wrk"
		dst := "dst/"
		c, err := NewCopyOperation(
			srcs, srcRoot, workDir, dst, validChown, pathutils.DefaultBlacklist, false)
		require.NoError(err)
		err = fs.addToLayer(newMemLayer(), c)
		require.NoError(err)

		n, err := findNode(fs, "/wrk", false, 0)
		require.NoError(err)
		require.NotNil(n)
		require.Equal("/wrk", n.dst)

		n, err = findNode(fs, "/wrk/dst", false, 0)
		require.NoError(err)
		require.NotNil(n)
		require.Equal("/wrk/dst", n.dst)

		n, err = findNode(fs, "/wrk/dst/test3.txt", false, 0)
		require.NoError(err)
		require.NotNil(n)
		require.Equal(tmpRoot+"/test1/test2/test3.txt", n.src)

		n, err = findNode(fs, "/wrk/dst/test5", false, 0)
		require.NoError(err)
		require.NotNil(n)
		require.Equal(tmpRoot+"/test1/test4/test5", n.src)

		n, err = findNode(fs, "/wrk/dst/test5/test6.txt", false, 0)
		require.NoError(err)
		require.NotNil(n)
		require.Equal(tmpRoot+"/test1/test4/test5/test6.txt", n.src)
	})
}

func TestAddLayerByScanWhiteout(t *testing.T) {
	require := require.New(t)

	tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
	require.NoError(err)
	defer os.RemoveAll(tmpRoot)

	clk := clock.NewMock()
	fs, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
	require.NoError(err)
	fs.blacklist = nil

	l := newMemLayer()
	dst11 := "/test1"
	require.NoError(addDirectoryToLayer(l, tmpRoot, dst11, 0755))
	dst12 := "/test1/test2"
	require.NoError(addDirectoryToLayer(l, tmpRoot, dst12, 0755))
	dst13 := "/test1/test2/test3.txt"
	require.NoError(addRegularFileToLayer(l, tmpRoot, dst13, "hello", 0755))
	dst14 := "/test1/test4"
	require.NoError(addDirectoryToLayer(l, tmpRoot, dst14, 0755))
	dst15 := "/test1/test4/test5"
	require.NoError(addDirectoryToLayer(l, tmpRoot, dst15, 0755))
	dst16 := "/test1/test4/test5/test6.txt"
	require.NoError(addRegularFileToLayer(l, tmpRoot, dst16, "hello", 0755))

	// Create layer by scan and commit.
	tarFile1, err := ioutil.TempFile("/tmp", "makisu-test-1.tar")
	defer os.Remove(tarFile1.Name())
	require.NoError(err)
	w1 := tar.NewWriter(tarFile1)
	err = fs.AddLayerByScan(w1)
	require.NoError(err)
	require.Equal(6, fs.layers[len(fs.layers)-1].count())
	w1.Close()
	tarFile1.Close()

	tarFile1, err = os.Open(tarFile1.Name())
	require.NoError(err)
	r := tar.NewReader(tarFile1)

	count := 0
	for {
		_, err := r.Next()
		if err == io.EOF {
			break
		}
		require.NoError(err)
		count++
	}
	require.Equal(6, count)

	// Remove existing file & scan again to create whiteout.
	os.RemoveAll(filepath.Join(tmpRoot, dst11))
	tarFile2, err := ioutil.TempFile("/tmp", "makisu-test-2.tar")
	defer os.Remove(tarFile2.Name())
	require.NoError(err)
	w2 := tar.NewWriter(tarFile2)
	err = fs.AddLayerByScan(w2)
	require.NoError(err)
	require.Equal(1, fs.layers[len(fs.layers)-1].count())
	w2.Close()
	tarFile2.Close()

	tarFile2, err = os.Open(tarFile2.Name())
	require.NoError(err)
	r = tar.NewReader(tarFile2)

	count = 0
	for {
		_, err := r.Next()
		if err == io.EOF {
			break
		}
		require.NoError(err)
		count++
	}
	require.Equal(1, count)
}

func TestAddLayersEqual(t *testing.T) {
	require := require.New(t)

	tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
	require.NoError(err)
	defer os.RemoveAll(tmpRoot)

	clk := clock.NewMock()
	fs1, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
	require.NoError(err)
	fs1.blacklist = nil

	fs2, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
	require.NoError(err)
	fs2.blacklist = nil

	fs3, err := NewMemFS(clk, tmpRoot, pathutils.DefaultBlacklist)
	require.NoError(err)
	fs3.blacklist = nil

	l := newMemLayer()
	dst11 := "/test1"
	require.NoError(addDirectoryToLayer(l, tmpRoot, dst11, 0755))
	dst12 := "/test1/test2"
	require.NoError(addDirectoryToLayer(l, tmpRoot, dst12, 0755))
	dst13 := "/test1/test2/test3.txt"
	require.NoError(addRegularFileToLayer(l, tmpRoot, dst13, "hello", 0755))
	dst14 := "/test1/test4"
	require.NoError(addDirectoryToLayer(l, tmpRoot, dst14, 0755))
	dst15 := "/test1/test4/test5"
	require.NoError(addDirectoryToLayer(l, tmpRoot, dst15, 0755))
	dst16 := "/test1/test4/test5/test6.txt"
	require.NoError(addRegularFileToLayer(l, tmpRoot, dst16, "hello", 0755))

	// Create layer by copy and commit.
	tarFile1, err := ioutil.TempFile("/tmp", "makisu-test-1.tar")
	defer os.Remove(tarFile1.Name())
	require.NoError(err)
	w1 := tar.NewWriter(tarFile1)
	srcs := []string{"/test1/test2/test3.txt", "/test1/test4"}
	srcRoot := tmpRoot
	workDir := "/wrk"
	dst := "dst/"
	c, err := NewCopyOperation(
		srcs, srcRoot, workDir, dst, validChown, pathutils.DefaultBlacklist, false)
	require.NoError(err)
	err = fs1.AddLayerByCopyOps([]*CopyOperation{c}, w1)
	require.NoError(err)
	w1.Close()

	// Create layer by scan and commit.
	tarFile2, err := ioutil.TempFile("/tmp", "makisu-test-2.tar")
	defer os.Remove(tarFile2.Name())
	require.NoError(err)
	w2 := tar.NewWriter(tarFile2)
	err = fs2.AddLayerByScan(w2)
	require.NoError(err)
	w2.Close()

	// Commit layer created using fixtures.
	tarFile3, err := ioutil.TempFile("/tmp", "makisu-test-3.tar")
	defer os.Remove(tarFile3.Name())
	require.NoError(err)
	w3 := tar.NewWriter(tarFile3)
	err = fs3.commitLayer(l, w3)
	require.NoError(err)
	w3.Close()

	// Check that all three resulting tarballs are the same.
	b1, err := ioutil.ReadAll(tarFile1)
	require.NoError(err)
	b2, err := ioutil.ReadAll(tarFile2)
	require.NoError(err)
	b3, err := ioutil.ReadAll(tarFile3)
	require.NoError(err)

	require.Equal(b3, b1)
	require.Equal(b2, b1)
}
