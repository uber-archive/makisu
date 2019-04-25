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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/uber/makisu/lib/pathutils"
	"github.com/uber/makisu/lib/utils/testutil"

	"github.com/stretchr/testify/require"
)

var (
	_hello     = []byte("hello")
	_hello2    = []byte("hello")
	validChown = fmt.Sprintf("%d:%d", testutil.CurrUID(), testutil.CurrGID())
)

func TestNewCopyOperation(t *testing.T) {
	require := require.New(t)

	tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
	require.NoError(err)
	defer os.RemoveAll(tmpRoot)

	srcs := []string{}
	srcRoot := "/srcRoot"
	workDir := ""
	dst := "/test2/test.txt"
	_, err = NewCopyOperation(
		srcs, srcRoot, workDir, dst, validChown, pathutils.DefaultBlacklist, false)
	require.Error(err)

	srcs = []string{"file", "dir/"}
	workDir = ""
	dst = "/target/test"
	_, err = NewCopyOperation(
		srcs, srcRoot, workDir, dst, validChown, pathutils.DefaultBlacklist, false)
	require.Error(err)

	srcs = []string{"file", "dir/"}
	workDir = ""
	dst = "target/test"
	_, err = NewCopyOperation(
		srcs, srcRoot, workDir, dst, validChown, pathutils.DefaultBlacklist, false)
	require.Error(err)

	srcs = []string{"file", "dir/"}
	workDir = "wrk/"
	dst = "target/test/"
	_, err = NewCopyOperation(
		srcs, srcRoot, workDir, dst, validChown, pathutils.DefaultBlacklist, false)
	require.Error(err)
}

func TestExecuteCopyOperation(t *testing.T) {
	tmpRoot1, err := ioutil.TempDir("/tmp", "makisu-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpRoot1)
	tmpRoot2, err := ioutil.TempDir("/tmp", "makisu-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpRoot2)

	t.Run("absolute file to absolute file", func(t *testing.T) {
		require := require.New(t)

		srcs := []string{"/test.txt"}
		require.NoError(ioutil.WriteFile(filepath.Join(tmpRoot1, "/test.txt"), _hello, os.ModePerm))
		require.NoError(os.Chown(filepath.Join(tmpRoot1, "/test.txt"), testutil.CurrUID(), testutil.CurrGID()))
		srcRoot := tmpRoot1
		dst := filepath.Join(tmpRoot2, "test2/test.txt")
		c, err := NewCopyOperation(
			srcs, srcRoot, "", dst, validChown, pathutils.DefaultBlacklist, false)
		require.NoError(err)
		require.NoError(c.Execute())
		b, err := ioutil.ReadFile(dst)
		require.NoError(err)
		require.Equal(_hello, b)
	})
	removeAllChildren(tmpRoot1, nil)
	removeAllChildren(tmpRoot2, nil)

	t.Run("absolute file to relative file", func(t *testing.T) {
		require := require.New(t)

		srcs := []string{"/test.txt"}
		require.NoError(ioutil.WriteFile(filepath.Join(tmpRoot1, "test.txt"), _hello, os.ModePerm))
		require.NoError(os.Chown(filepath.Join(tmpRoot1, "/test.txt"), testutil.CurrUID(), testutil.CurrGID()))
		srcRoot := tmpRoot1
		workDir := tmpRoot2
		dst := "test2/test.txt"
		c, err := NewCopyOperation(
			srcs, srcRoot, workDir, dst, validChown, pathutils.DefaultBlacklist, false)
		require.NoError(err)
		require.NoError(c.Execute())
		b, err := ioutil.ReadFile(filepath.Join(tmpRoot2, dst))
		require.NoError(err)
		require.Equal(_hello, b)
	})
	removeAllChildren(tmpRoot1, nil)
	removeAllChildren(tmpRoot2, nil)

	t.Run("absolute files to absolute dir", func(t *testing.T) {
		require := require.New(t)

		srcs := []string{"/test.txt", "/test2.txt"}
		require.NoError(ioutil.WriteFile(filepath.Join(tmpRoot1, "test.txt"), _hello, os.ModePerm))
		require.NoError(os.Chown(filepath.Join(tmpRoot1, "test.txt"), testutil.CurrUID(), testutil.CurrGID()))
		require.NoError(ioutil.WriteFile(filepath.Join(tmpRoot1, "test2.txt"), _hello2, os.ModePerm))
		require.NoError(os.Chown(filepath.Join(tmpRoot1, "test2.txt"), testutil.CurrUID(), testutil.CurrGID()))
		srcRoot := tmpRoot1
		workDir := tmpRoot2
		dst := "test2/"
		c, err := NewCopyOperation(
			srcs, srcRoot, workDir, dst, validChown, pathutils.DefaultBlacklist, false)
		require.NoError(err)
		require.NoError(c.Execute())
		b, err := ioutil.ReadFile(filepath.Join(tmpRoot2, dst, "test.txt"))
		require.NoError(err)
		require.Equal(_hello, b)
		b, err = ioutil.ReadFile(filepath.Join(tmpRoot2, dst, "test2.txt"))
		require.NoError(err)
		require.Equal(_hello2, b)
	})
	removeAllChildren(tmpRoot1, nil)
	removeAllChildren(tmpRoot2, nil)

	t.Run("absolute files to relative dir", func(t *testing.T) {
		require := require.New(t)

		srcs := []string{"/test.txt", "/test2.txt"}
		require.NoError(ioutil.WriteFile(filepath.Join(tmpRoot1, "test.txt"), _hello, os.ModePerm))
		require.NoError(os.Chown(filepath.Join(tmpRoot1, "test.txt"), testutil.CurrUID(), testutil.CurrGID()))
		require.NoError(ioutil.WriteFile(filepath.Join(tmpRoot1, "test2.txt"), _hello2, os.ModePerm))
		require.NoError(os.Chown(filepath.Join(tmpRoot1, "test2.txt"), testutil.CurrUID(), testutil.CurrGID()))
		srcRoot := tmpRoot1
		workDir := filepath.Join(tmpRoot2, "test2")
		dst := "."
		c, err := NewCopyOperation(
			srcs, srcRoot, workDir, dst, validChown, pathutils.DefaultBlacklist, false)
		require.NoError(err)
		require.NoError(c.Execute())
		b, err := ioutil.ReadFile(filepath.Join(tmpRoot2, "test2", "test.txt"))
		require.NoError(err)
		require.Equal(_hello, b)
		b, err = ioutil.ReadFile(filepath.Join(tmpRoot2, "test2", "test2.txt"))
		require.NoError(err)
		require.Equal(_hello2, b)
	})
	removeAllChildren(tmpRoot1, nil)
	removeAllChildren(tmpRoot2, nil)

	t.Run("absolute dirs to relative dir", func(t *testing.T) {
		require := require.New(t)

		srcs := []string{"/test/", "/test2/"}
		require.NoError(os.MkdirAll(filepath.Join(tmpRoot1, "test"), os.ModePerm))
		require.NoError(os.MkdirAll(filepath.Join(tmpRoot1, "test2"), os.ModePerm))
		require.NoError(ioutil.WriteFile(filepath.Join(tmpRoot1, "test", "test.txt"), _hello, os.ModePerm))
		require.NoError(os.Chown(filepath.Join(tmpRoot1, "test", "test.txt"), testutil.CurrUID(), testutil.CurrGID()))
		require.NoError(ioutil.WriteFile(filepath.Join(tmpRoot1, "test2", "test2.txt"), _hello2, os.ModePerm))
		require.NoError(os.Chown(filepath.Join(tmpRoot1, "test2", "test2.txt"), testutil.CurrUID(), testutil.CurrGID()))
		srcRoot := tmpRoot1
		workDir := tmpRoot2
		dst := "test2/"
		c, err := NewCopyOperation(
			srcs, srcRoot, workDir, dst, validChown, pathutils.DefaultBlacklist, false)
		require.NoError(err)
		require.NoError(c.Execute())
		b, err := ioutil.ReadFile(filepath.Join(tmpRoot2, dst, "test.txt"))
		require.NoError(err)
		require.Equal(_hello, b)
		b, err = ioutil.ReadFile(filepath.Join(tmpRoot2, dst, "test2.txt"))
		require.NoError(err)
		require.Equal(_hello2, b)
	})
	removeAllChildren(tmpRoot1, nil)
	removeAllChildren(tmpRoot2, nil)

	t.Run("absolute dir and file to relative dir", func(t *testing.T) {
		require := require.New(t)

		srcs := []string{"/test/", "/test2.txt"}
		require.NoError(os.MkdirAll(filepath.Join(tmpRoot1, "test"), os.ModePerm))
		require.NoError(ioutil.WriteFile(filepath.Join(tmpRoot1, "test", "test.txt"), _hello, os.ModePerm))
		require.NoError(os.Chown(filepath.Join(tmpRoot1, "test", "test.txt"), testutil.CurrUID(), testutil.CurrGID()))
		require.NoError(ioutil.WriteFile(filepath.Join(tmpRoot1, "test2.txt"), _hello2, os.ModePerm))
		require.NoError(os.Chown(filepath.Join(tmpRoot1, "test2.txt"), testutil.CurrUID(), testutil.CurrGID()))
		srcRoot := tmpRoot1
		workDir := tmpRoot2
		dst := "test2/"
		c, err := NewCopyOperation(
			srcs, srcRoot, workDir, dst, validChown, pathutils.DefaultBlacklist, false)
		require.NoError(err)
		require.NoError(c.Execute())
		b, err := ioutil.ReadFile(filepath.Join(tmpRoot2, dst, "test.txt"))
		require.NoError(err)
		require.Equal(_hello, b)
		b, err = ioutil.ReadFile(filepath.Join(tmpRoot2, dst, "test2.txt"))
		require.NoError(err)
		require.Equal(_hello2, b)
	})
	removeAllChildren(tmpRoot1, nil)
	removeAllChildren(tmpRoot2, nil)
}
