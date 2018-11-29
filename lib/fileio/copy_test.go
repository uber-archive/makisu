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

package fileio

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/uber/makisu/lib/pathutils"
	"github.com/uber/makisu/lib/utils"

	"github.com/stretchr/testify/require"
)

var currUID int
var currGID int

func init() {
	var err error
	currUID, currGID, err = utils.GetUIDGID()
	if err != nil {
		panic(err)
	}
}

func TestCopyFileDanglingSymlink(t *testing.T) {
	require := require.New(t)

	sourceDir, err := ioutil.TempDir("/tmp", "testCopy")
	require.NoError(err)
	defer os.RemoveAll(sourceDir)
	sourceSymlink := filepath.Join(sourceDir, "link")
	require.NoError(os.Symlink("/nonexistent", sourceSymlink))

	target, err := ioutil.TempFile("/tmp", "testCopy")
	require.NoError(err)
	defer os.Remove(target.Name())
	defer target.Close()

	c := NewCopier(pathutils.DefaultBlacklist)
	require.NoError(c.CopyFile(sourceSymlink, target.Name(), currUID, currGID))

	result, err := os.Readlink(target.Name())
	require.NoError(err)
	require.Equal("/nonexistent", result)
}

func TestCopyFileTargetNotExist(t *testing.T) {
	require := require.New(t)

	source, err := ioutil.TempFile("/tmp", "testCopy")
	require.NoError(err)
	defer os.Remove(source.Name())
	defer source.Close()
	target, err := ioutil.TempFile("/tmp", "testCopy")
	require.NoError(err)
	target.Close()
	os.Remove(target.Name())
	defer os.Remove(target.Name())

	testString := "Testing COPY"
	_, err = source.WriteString(testString)
	require.NoError(err)

	c := NewCopier(pathutils.DefaultBlacklist)
	require.NoError(c.CopyFile(source.Name(), target.Name(), currUID, currGID))

	result, err := ioutil.ReadFile(target.Name())
	require.NoError(err)
	require.Equal(testString, string(result))
}

func TestCopyFileSetSpecialBit(t *testing.T) {
	require := require.New(t)

	source, err := ioutil.TempFile("/tmp", "testCopy")
	require.NoError(err)
	defer os.Remove(source.Name())
	defer source.Close()
	testString := "Testing COPY"
	_, err = source.WriteString(testString)
	require.NoError(err)
	// Set setuid bit on source.
	require.NoError(os.Chmod(source.Name(), os.ModePerm|os.ModeSetuid))

	target, err := ioutil.TempFile("/tmp", "testCopy")
	require.NoError(err)
	target.Close()
	os.Remove(target.Name())
	defer os.Remove(target.Name())

	c := NewCopier(pathutils.DefaultBlacklist)
	require.NoError(c.CopyFile(source.Name(), target.Name(), currUID, currGID))

	result, err := ioutil.ReadFile(target.Name())
	require.NoError(err)
	require.Equal(testString, string(result))
	targetFi, err := os.Stat(target.Name())
	require.NoError(err)
	require.Equal(targetFi.Mode(), os.ModePerm|os.ModeSetuid)
}

func TestCopyFileTargetEmpty(t *testing.T) {
	require := require.New(t)

	source, err := ioutil.TempFile("/tmp", "testCopy")
	require.NoError(err)
	defer os.Remove(source.Name())
	defer source.Close()
	target, err := ioutil.TempFile("/tmp", "testCopy")
	require.NoError(err)
	defer os.Remove(target.Name())
	defer target.Close()

	testString := "Testing COPY"
	_, err = source.WriteString(testString)
	require.NoError(err)

	c := NewCopier(pathutils.DefaultBlacklist)
	require.NoError(c.CopyFile(source.Name(), target.Name(), currUID, currGID))

	result, err := ioutil.ReadFile(target.Name())
	require.NoError(err)
	require.Equal(testString, string(result))
}

func TestCopyFileTargetOverwrite(t *testing.T) {
	require := require.New(t)

	source, err := ioutil.TempFile("/tmp", "testCopy")
	require.NoError(err)
	defer os.Remove(source.Name())
	defer source.Close()
	target, err := ioutil.TempFile("/tmp", "testCopy")
	require.NoError(err)
	defer os.Remove(target.Name())
	defer target.Close()

	testString := "Testing COPY"
	_, err = source.WriteString(testString)
	require.NoError(err)
	_, err = target.WriteString("To be overwritten")
	require.NoError(err)

	c := NewCopier(pathutils.DefaultBlacklist)
	require.NoError(c.CopyFile(source.Name(), target.Name(), currUID, currGID))

	result, err := ioutil.ReadFile(target.Name())
	require.NoError(err)
	require.Equal(testString, string(result))
}

func TestCopyDirectoryTargetNotExist(t *testing.T) {
	require := require.New(t)

	sourceDir, err := ioutil.TempDir("/tmp", "testCopy")
	require.NoError(err)
	defer os.RemoveAll(sourceDir)
	targetDir, err := ioutil.TempDir("/tmp", "testCopyTargetDir")
	require.NoError(err)
	defer os.RemoveAll(targetDir)

	sourceSubDirOne, err := ioutil.TempDir(sourceDir, "testCopy")
	require.NoError(err)
	sourceSubDirTwo, err := ioutil.TempDir(sourceDir, "testCopy")
	require.NoError(err)
	sourceFileOne, err := ioutil.TempFile(sourceSubDirOne, "testCopy")
	require.NoError(err)
	defer sourceFileOne.Close()
	_, err = sourceFileOne.WriteString("Test source file one")
	require.NoError(err)
	sourceFileTwo, err := ioutil.TempFile(sourceDir, "testCopy")
	require.NoError(err)
	defer sourceFileTwo.Close()
	_, err = sourceFileTwo.WriteString("Test source file two")
	require.NoError(err)

	// Perform copy.
	c := NewCopier(pathutils.DefaultBlacklist)
	require.NoError(c.CopyDir(sourceDir, targetDir, currUID, currGID))

	// Verify.
	_, err = os.Stat(path.Join(targetDir, path.Base(sourceSubDirOne)))
	require.NoError(err)
	_, err = os.Stat(path.Join(targetDir, path.Base(sourceSubDirTwo)))
	require.NoError(err)
	resultOne, err := ioutil.ReadFile(path.Join(targetDir, path.Base(sourceSubDirOne), path.Base(sourceFileOne.Name())))
	require.NoError(err)
	require.Equal("Test source file one", string(resultOne))
	resultTwo, err := ioutil.ReadFile(path.Join(targetDir, path.Base(sourceFileTwo.Name())))
	require.NoError(err)
	require.Equal("Test source file two", string(resultTwo))
}

func TestCopyDirectoryTargetExists(t *testing.T) {
	require := require.New(t)

	sourceDir, err := ioutil.TempDir("/tmp", "testCopy")
	require.NoError(err)
	defer os.RemoveAll(sourceDir)
	targetDir, err := ioutil.TempDir("/tmp", "testCopyTargetDir")
	require.NoError(err)
	defer os.RemoveAll(targetDir)

	sourceSubDirOne, err := ioutil.TempDir(sourceDir, "testCopy")
	require.NoError(err)
	sourceSubDirTwo, err := ioutil.TempDir(sourceDir, "testCopy")
	require.NoError(err)
	sourceFileOne, err := ioutil.TempFile(sourceSubDirOne, "testCopy")
	require.NoError(err)
	defer sourceFileOne.Close()
	_, err = sourceFileOne.WriteString("Test source file one")
	require.NoError(err)
	sourceFileTwo, err := ioutil.TempFile(sourceDir, "testCopy")
	require.NoError(err)
	defer sourceFileTwo.Close()
	_, err = sourceFileTwo.WriteString("Test source file two")
	require.NoError(err)

	// Create one existing file in target dir.
	targetFileOne, err := ioutil.TempFile(targetDir, "testCopy")
	require.NoError(err)
	defer targetFileOne.Close()
	_, err = targetFileOne.WriteString("Test target file one")
	require.NoError(err)

	// Perform copy.
	c := NewCopier(pathutils.DefaultBlacklist)
	require.NoError(c.CopyDir(sourceDir, targetDir, currUID, currGID))

	// Verify.
	_, err = os.Stat(path.Join(targetDir, path.Base(sourceSubDirOne)))
	require.NoError(err)
	_, err = os.Stat(path.Join(targetDir, path.Base(sourceSubDirTwo)))
	require.NoError(err)
	resultOne, err := ioutil.ReadFile(path.Join(targetDir, path.Base(sourceSubDirOne), path.Base(sourceFileOne.Name())))
	require.NoError(err)
	require.Equal("Test source file one", string(resultOne))
	resultTwo, err := ioutil.ReadFile(path.Join(targetDir, path.Base(sourceFileTwo.Name())))
	require.NoError(err)
	require.Equal("Test source file two", string(resultTwo))
	resultTargetOne, err := ioutil.ReadFile(targetFileOne.Name())
	require.NoError(err)
	require.Equal("Test target file one", string(resultTargetOne))
}

func TestCopyDirectoryIncludingSymlink(t *testing.T) {
	require := require.New(t)

	sourceDir, err := ioutil.TempDir("/tmp", "testCopy")
	require.NoError(err)
	defer os.RemoveAll(sourceDir)
	targetDir, err := ioutil.TempDir("/tmp", "testCopyTargetDir")
	require.NoError(err)
	defer os.RemoveAll(targetDir)

	sourceSubDirOne, err := ioutil.TempDir(sourceDir, "testCopy")
	require.NoError(err)
	sourceSymlinkOne := filepath.Join(sourceDir, "link")
	require.NoError(os.Symlink(sourceSubDirOne, sourceSymlinkOne))
	sourceFileOne, err := ioutil.TempFile(sourceSubDirOne, "testCopy")
	require.NoError(err)
	defer sourceFileOne.Close()
	_, err = sourceFileOne.WriteString("Test source file one")
	require.NoError(err)
	sourceFileTwo, err := ioutil.TempFile(sourceDir, "testCopy")
	require.NoError(err)
	defer sourceFileTwo.Close()
	_, err = sourceFileTwo.WriteString("Test source file two")
	require.NoError(err)

	// Perform copy.
	c := NewCopier(pathutils.DefaultBlacklist)
	require.NoError(c.CopyDir(sourceDir, targetDir, currUID, currGID))

	// Verify.
	_, err = os.Stat(path.Join(targetDir, path.Base(sourceSubDirOne)))
	require.NoError(err)
	_, err = os.Stat(path.Join(targetDir, path.Base(sourceSymlinkOne)))
	require.NoError(err)
	linkTarget, err := os.Readlink(path.Join(targetDir, path.Base(sourceSymlinkOne)))
	require.NoError(err)
	require.Equal(sourceSubDirOne, linkTarget)
	resultOne, err := ioutil.ReadFile(path.Join(targetDir, path.Base(sourceSubDirOne), path.Base(sourceFileOne.Name())))
	require.NoError(err)
	require.Equal("Test source file one", string(resultOne))
	resultTwo, err := ioutil.ReadFile(path.Join(targetDir, path.Base(sourceFileTwo.Name())))
	require.NoError(err)
	require.Equal("Test source file two", string(resultTwo))
}

func TestCopyDirectoryInfiniteLoop(t *testing.T) {
	require := require.New(t)

	// Make targetDir child of source, creating infinite loop.
	sourceDir, err := ioutil.TempDir("/tmp", "testCopy")
	require.NoError(err)
	defer os.RemoveAll(sourceDir)
	targetDir, err := ioutil.TempDir(sourceDir, "testCopyTargetDir")
	require.NoError(err)
	defer os.RemoveAll(targetDir)

	sourceSubDirOne, err := ioutil.TempDir(sourceDir, "testCopy")
	require.NoError(err)
	sourceSymlinkOne := filepath.Join(sourceDir, "link")
	require.NoError(os.Symlink(sourceSubDirOne, sourceSymlinkOne))
	sourceFileOne, err := ioutil.TempFile(sourceSubDirOne, "testCopy")
	require.NoError(err)
	defer sourceFileOne.Close()
	_, err = sourceFileOne.WriteString("Test source file one")
	require.NoError(err)
	sourceFileTwo, err := ioutil.TempFile(sourceDir, "testCopy")
	require.NoError(err)
	defer sourceFileTwo.Close()
	_, err = sourceFileTwo.WriteString("Test source file two")
	require.NoError(err)

	// Perform copy.
	c := NewCopier(pathutils.DefaultBlacklist)
	require.NoError(c.CopyDir(sourceDir, targetDir, currUID, currGID))

	// Verify.
	_, err = os.Stat(path.Join(targetDir, path.Base(sourceSubDirOne)))
	require.NoError(err)
	_, err = os.Stat(path.Join(targetDir, path.Base(sourceSymlinkOne)))
	require.NoError(err)
	linkTarget, err := os.Readlink(path.Join(targetDir, path.Base(sourceSymlinkOne)))
	require.NoError(err)
	require.Equal(sourceSubDirOne, linkTarget)
	resultOne, err := ioutil.ReadFile(path.Join(targetDir, path.Base(sourceSubDirOne), path.Base(sourceFileOne.Name())))
	require.NoError(err)
	require.Equal("Test source file one", string(resultOne))
	resultTwo, err := ioutil.ReadFile(path.Join(targetDir, path.Base(sourceFileTwo.Name())))
	require.NoError(err)
	require.Equal("Test source file two", string(resultTwo))

	// TargetDir was not recreated.
	_, err = os.Stat(path.Join(targetDir, path.Base(targetDir)))
	require.True(os.IsNotExist(err))
}
