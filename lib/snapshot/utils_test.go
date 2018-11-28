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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveHardLink(t *testing.T) {
	require := require.New(t)

	tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
	require.NoError(err)
	defer os.RemoveAll(tmpRoot)

	tmp1, err := ioutil.TempFile(tmpRoot, "test1")
	require.NoError(err)
	fi1, err := os.Lstat(tmp1.Name())
	require.NoError(err)
	tmp2 := filepath.Join(tmpRoot, "link2")
	require.NoError(os.Link(tmp1.Name(), tmp2))
	fi2, err := os.Lstat(tmp2)
	require.NoError(err)
	tmp3 := filepath.Join(tmpRoot, "link3")
	require.NoError(os.Link(tmp1.Name(), tmp3))
	fi3, err := os.Lstat(tmp3)
	require.NoError(err)

	inode1 := resolveHardLink(tmp1.Name(), fi1)
	inode2 := resolveHardLink(tmp2, fi2)
	require.Equal(inode1, inode2)
	inode3 := resolveHardLink(tmp3, fi3)
	require.Equal(inode2, inode3)
}

func TestResolveSymlink(t *testing.T) {
	require := require.New(t)

	tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
	require.NoError(err)
	defer os.RemoveAll(tmpRoot)

	tmp1, err := ioutil.TempFile(tmpRoot, "test1")
	require.NoError(err)
	fi1, err := os.Lstat(tmp1.Name())
	require.NoError(err)
	tmp2 := filepath.Join(tmpRoot, "link2")
	require.NoError(os.Symlink(tmp1.Name(), tmp2))
	fi2, err := os.Lstat(tmp2)
	require.NoError(err)
	tmp3 := filepath.Join(tmpRoot, "link3")
	require.NoError(os.Symlink(tmp1.Name(), tmp3))
	fi3, err := os.Lstat(tmp3)
	require.NoError(err)

	ok, _, err := resolveSymlink(tmp1.Name(), fi1)
	require.NoError(err)
	require.False(ok)
	ok, target2, err := resolveSymlink(tmp2, fi2)
	require.NoError(err)
	require.True(ok)
	require.Equal(tmp1.Name(), target2)
	ok, target3, err := resolveSymlink(tmp3, fi3)
	require.NoError(err)
	require.True(ok)
	require.Equal(tmp1.Name(), target3)
}

func TestEvalSymlink(t *testing.T) {
	t.Run("no_symlinks", func(t *testing.T) {
		require := require.New(t)
		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		dir1 := filepath.Join(tmpRoot, "dir1")
		require.NoError(os.Mkdir(dir1, os.ModePerm))
		tmp1 := filepath.Join(tmpRoot, "dir1", "tmp1")
		_, err = os.Create(tmp1)
		require.NoError(err)

		path, err := evalSymlinks(filepath.Join("dir1", "tmp1"), tmpRoot)
		require.NoError(err)
		require.Equal(filepath.Join("/", "dir1", "tmp1"), path)
	})

	t.Run("simple_case", func(t *testing.T) {
		require := require.New(t)
		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		tmp1 := filepath.Join(tmpRoot, "test1")
		_, err = os.Create(tmp1)
		require.NoError(err)
		tmp2 := filepath.Join(tmpRoot, "link2")
		require.NoError(os.Symlink("test1", tmp2))
		tmp3 := filepath.Join(tmpRoot, "link3")
		require.NoError(os.Symlink(tmp2, tmp3))

		path, err := evalSymlinks("link2", tmpRoot)
		require.NoError(err)
		require.Equal("/test1", path)
		path, err = evalSymlinks("link3", tmpRoot)
		require.NoError(err)
		require.Equal("/test1", path)
	})

	t.Run("layered", func(t *testing.T) {
		require := require.New(t)
		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		dir1 := filepath.Join(tmpRoot, "dir1")
		require.NoError(os.Mkdir(dir1, os.ModePerm))
		tmp1 := filepath.Join(tmpRoot, "dir1", "tmp1")
		_, err = os.Create(tmp1)
		require.NoError(err)

		dir2 := filepath.Join(tmpRoot, "dir2")
		require.NoError(os.Mkdir(dir2, os.ModePerm))
		dir3 := filepath.Join(tmpRoot, "dir2", "dir3")
		require.NoError(os.Symlink(dir1, dir3))

		path, err := evalSymlinks(filepath.Join("dir2", "dir3", "tmp1"), tmpRoot)
		require.NoError(err)
		require.Equal("/dir1/tmp1", path)
	})
}

func TestRemoveAll(t *testing.T) {
	require := require.New(t)

	tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
	require.NoError(err)
	defer os.RemoveAll(tmpRoot)

	tmp1, err := ioutil.TempFile(tmpRoot, "test1")
	require.NoError(err)
	_, err = os.Lstat(tmp1.Name())
	require.NoError(err)
	tmp2 := filepath.Join(tmpRoot, "tmp2")
	require.NoError(os.Symlink(tmp1.Name(), tmp2))
	_, err = os.Lstat(tmp2)
	require.NoError(err)
	dir1 := filepath.Join(tmpRoot, "dir1")
	err = os.Mkdir(dir1, os.ModePerm)
	require.NoError(err)
	tmp3, err := ioutil.TempFile(dir1, "test3")
	require.NoError(err)
	dir2 := filepath.Join(tmpRoot, "dir2")
	err = os.Mkdir(dir2, os.ModePerm)
	require.NoError(err)
	tmp4, err := ioutil.TempFile(dir2, "test4")
	require.NoError(err)

	require.NoError(removeAllChildren(tmpRoot, []string{dir2}))

	_, err = os.Lstat(tmp1.Name())
	require.True(os.IsNotExist(err))
	_, err = os.Lstat(tmp2)
	require.True(os.IsNotExist(err))
	_, err = os.Lstat(dir1)
	require.True(os.IsNotExist(err))
	_, err = os.Lstat(tmp3.Name())
	require.True(os.IsNotExist(err))
	_, err = os.Lstat(dir2)
	require.NoError(err)
	_, err = os.Lstat(tmp4.Name())
	require.NoError(err)
}
