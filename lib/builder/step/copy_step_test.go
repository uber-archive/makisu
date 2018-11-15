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

package step

import (
	"archive/tar"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/storage"
	"github.com/uber/makisu/lib/tario"

	"github.com/stretchr/testify/require"
)

func abs(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return absPath
}

func TestNewCopyStep(t *testing.T) {
	require := require.New(t)

	_, err := NewCopyStep("", validChown, "", []string{"src", "src"}, "dst", false)
	require.Error(err)
}

func TestCopyStepSetCacheID(t *testing.T) {
	t.Run("CopyFromSameContext", func(t *testing.T) {
		require := require.New(t)
		context, cleanup := context.BuildContextFixture()
		defer cleanup()

		sourceDir, err := ioutil.TempDir(context.ContextDir, "testCopyStepSource")
		require.NoError(err)
		sourceSubDirOne, err := ioutil.TempDir(sourceDir, "testCopyStepSub")
		require.NoError(err)
		sourceFileOne, err := ioutil.TempFile(sourceSubDirOne, "testCopyStepFile")
		require.NoError(err)

		rand.Seed(time.Now().UnixNano())
		sourceFileOneContent := make([]byte, 1024)
		rand.Read(sourceFileOneContent)
		_, err = sourceFileOne.Write(sourceFileOneContent)
		require.NoError(err)
		defer sourceFileOne.Close()

		step := CopyStepFixture("", "", []string{"."}, "tmp", false)
		err = step.SetCacheID(context, "")
		hash1 := step.CacheID()
		require.NoError(err)

		err = step.SetCacheID(context, "")
		require.NoError(err)

		require.Equal(hash1, step.CacheID())
	})

	t.Run("CopyFromSameContextDifferentSeed", func(t *testing.T) {
		require := require.New(t)
		context, cleanup := context.BuildContextFixture()
		defer cleanup()

		sourceDir, err := ioutil.TempDir(context.ContextDir, "testCopyStepSource")
		require.NoError(err)
		sourceSubDirOne, err := ioutil.TempDir(sourceDir, "testCopyStepSub")
		require.NoError(err)
		sourceFileOne, err := ioutil.TempFile(sourceSubDirOne, "testCopyStepFile")
		require.NoError(err)

		rand.Seed(time.Now().UnixNano())
		sourceFileOneContent := make([]byte, 1024)
		rand.Read(sourceFileOneContent)
		_, err = sourceFileOne.Write(sourceFileOneContent)
		require.NoError(err)
		defer sourceFileOne.Close()

		step := CopyStepFixture("", "", []string{"."}, "tmp", false)
		err = step.SetCacheID(context, "")
		hash1 := step.CacheID()
		require.NoError(err)

		err = step.SetCacheID(context, hash1)
		require.NoError(err)

		require.NotEqual(hash1, step.CacheID())
	})

	t.Run("CopyFromDifferentContexts", func(t *testing.T) {
		require := require.New(t)
		context, cleanup := context.BuildContextFixture()
		defer cleanup()

		sourceDir, err := ioutil.TempDir(context.ContextDir, "testCopyStepSource")
		require.NoError(err)
		sourceSubDirOne, err := ioutil.TempDir(sourceDir, "testCopyStepSub")
		require.NoError(err)
		sourceFileOne, err := ioutil.TempFile(sourceSubDirOne, "testCopyStepFile")
		require.NoError(err)

		rand.Seed(time.Now().UnixNano())
		sourceFileOneContent := make([]byte, 1024)
		rand.Read(sourceFileOneContent)
		_, err = sourceFileOne.Write(sourceFileOneContent)
		require.NoError(err)
		defer sourceFileOne.Close()

		step := CopyStepFixture("", "", []string{"."}, "tmp", false)
		err = step.SetCacheID(context, "")
		hash1 := step.CacheID()
		require.NoError(err)

		step2 := CopyStepFixture("", "", []string{"."}, "tmp2", false)
		err = step2.SetCacheID(context, hash1)
		require.NoError(err)

		// Hash should be different because the destination changes.
		require.NotEqual(hash1, step2.CacheID())

		require.NoError(ioutil.WriteFile(sourceFileOne.Name(), []byte("new content"), 0755))
		err = step.SetCacheID(context, step.CacheID())
		require.NoError(err)

		// Hash should be different because content in context is changed.
		require.NotEqual(hash1, step.CacheID())
	})

	t.Run("CopyFromStage", func(t *testing.T) {
		require := require.New(t)
		context, cleanup := context.BuildContextFixture()
		defer cleanup()

		step := CopyStepFixture("", "stage", []string{"."}, "tmp", false)
		err := step.SetCacheID(context, "")
		hash1 := step.CacheID()
		require.NoError(err)

		err = step.SetCacheID(context, hash1)
		hash2 := step.CacheID()
		require.NoError(err)

		// hash1 and hash2 should be generated randomly.
		require.NotEqual(hash1, hash2)
	})
}

func TestCopyStepExecuteOnCriticalPath(t *testing.T) {
	store, cleanup := storage.StoreFixture()
	defer cleanup()

	// Create temp test dirs and files in context dir.
	contextDir, err := ioutil.TempDir("./", "testCopyStepContextDir")
	require.NoError(t, err)
	defer os.RemoveAll(contextDir)

	sandboxDir, err := ioutil.TempDir("./", "testCopyStepSandboxDir")
	require.NoError(t, err)
	defer os.RemoveAll(sandboxDir)

	context := &context.BuildContext{
		ContextDir: contextDir,
		ImageStore: store,
	}

	sourceDir, err := ioutil.TempDir(contextDir, "testCopyStep")
	require.NoError(t, err)
	sourceSubDirOne, err := ioutil.TempDir(sourceDir, "testCopyStep")
	require.NoError(t, err)
	sourceSubDirTwo, err := ioutil.TempDir(sourceDir, "testCopyStep")
	require.NoError(t, err)
	sourceFileOne, err := ioutil.TempFile(sourceSubDirOne, "testCopyStep")
	require.NoError(t, err)
	defer sourceFileOne.Close()
	_, err = sourceFileOne.WriteString("Test source file one")
	sourceFileTwo, err := ioutil.TempFile(sourceDir, "testCopyStep")
	require.NoError(t, err)
	defer sourceFileTwo.Close()
	_, err = sourceFileTwo.WriteString("Test source file two")

	t.Run("OneSourceFileToTargetFile", func(t *testing.T) {
		require := require.New(t)
		sourceFileOneRelPath, err := filepath.Rel(contextDir, sourceFileOne.Name())
		require.NoError(err)

		targetPath := "./testCopyStepExecuteOnCriticalPath_OneSourceFileToTargetFile"
		defer os.Remove(targetPath)

		srcs := []string{sourceFileOneRelPath}
		step := CopyStepFixture("", "", srcs, abs(targetPath), false)
		err = step.Execute(context, true)
		require.NoError(err)

		// Verify file is at local path.
		result, err := ioutil.ReadFile(targetPath)
		require.NoError(err)
		require.Equal("Test source file one", string(result))
	})

	t.Run("OneSourceFileToTargetDir", func(t *testing.T) {
		require := require.New(t)
		sourceFileOneRelPath, err := filepath.Rel(contextDir, sourceFileOne.Name())
		require.NoError(err)

		targetPath := "./testCopyStepExecuteOnCriticalPath_OneSourceFileToTargetDir"
		defer os.RemoveAll(targetPath)

		srcs := []string{sourceFileOneRelPath}
		step := CopyStepFixture("", "", srcs, abs(targetPath)+"/", false)
		err = step.Execute(context, true)
		require.NoError(err)

		// Verify file is at local path.
		result, err := ioutil.ReadFile(path.Join(targetPath, path.Base(sourceFileOne.Name())))
		require.NoError(err)
		require.Equal("Test source file one", string(result))
	})

	t.Run("MultipleSourceFilesToTargetDir", func(t *testing.T) {
		require := require.New(t)
		sourceFileOneRelPath, err := filepath.Rel(contextDir, sourceFileOne.Name())
		require.NoError(err)
		sourceFileTwoRelPath, err := filepath.Rel(contextDir, sourceFileTwo.Name())
		require.NoError(err)

		targetPath := "./testCopyStepExecuteOnCriticalPath_MultipleSourceFilesToTargetDir"
		defer os.RemoveAll(targetPath)

		srcs := []string{sourceFileOneRelPath, sourceFileTwoRelPath}
		step := CopyStepFixture("", "", srcs, abs(targetPath)+"/", false)
		err = step.Execute(context, true)
		require.NoError(err)

		// Verify file is at local path.
		resultOne, err := ioutil.ReadFile(path.Join(targetPath, path.Base(sourceFileOne.Name())))
		require.NoError(err)
		require.Equal("Test source file one", string(resultOne))
		resultTwo, err := ioutil.ReadFile(path.Join(targetPath, path.Base(sourceFileTwo.Name())))
		require.NoError(err)
		require.Equal("Test source file two", string(resultTwo))
	})

	t.Run("OneSourceDirWithTrailingSlash", func(t *testing.T) {
		require := require.New(t)
		targetDir := "./testCopyStepExecuteOnCriticalPath_OneSourceDirWithTrailingSlash"
		require.NoError(err)
		defer os.RemoveAll(targetDir)

		// Copy to local path.
		srcs := []string{path.Base(sourceDir)}
		step := CopyStepFixture("", "", srcs, abs(targetDir)+"/", false)
		err = step.Execute(context, true)
		require.NoError(err)

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
	})

	t.Run("OneSourceDirWithoutTrailingSlash", func(t *testing.T) {
		require := require.New(t)
		targetDir, err := ioutil.TempDir("./", "testCopyStepTarget")
		require.NoError(err)
		defer os.RemoveAll(targetDir)

		// Copy to local path.
		srcs := []string{path.Base(sourceDir)}
		step := CopyStepFixture("", "", srcs, abs(targetDir), false)
		err = step.Execute(context, true)
		require.NoError(err)

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
	})

	t.Run("MultipleSourceDirs", func(t *testing.T) {
		require := require.New(t)
		sourceDirOneRelPath, err := filepath.Rel(contextDir, sourceSubDirOne)
		require.NoError(err)

		targetDir, err := ioutil.TempDir("./", "testCopyStepTarget")
		require.NoError(err)
		defer os.RemoveAll(targetDir)

		// Copy to local path.
		srcs := []string{path.Base(sourceDir), sourceDirOneRelPath}
		step := CopyStepFixture("", "", srcs, abs(targetDir)+"/", false)
		err = step.Execute(context, true)
		require.NoError(err)

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

		// File one was also copied from sourceDirOneRelPath to target root.
		resultThree, err := ioutil.ReadFile(path.Join(targetDir, path.Base(sourceFileOne.Name())))
		require.NoError(err)
		require.Equal("Test source file one", string(resultThree))
	})
}

func TestCopyStepCommitOnNonCriticalPath(t *testing.T) {
	t.Run("OneSourceFileToTargetFile", func(t *testing.T) {
		require := require.New(t)
		context, cleanup := context.BuildContextFixture()
		// This is only used for the path inside the tarball,
		// we do not mutate / in actual fs.
		workingDir := "/"
		defer cleanup()

		sourceDir, err := ioutil.TempDir(context.ContextDir, "testCopyStepSource")
		require.NoError(err)
		sourceSubDirOne, err := ioutil.TempDir(sourceDir, "testCopyStepSub")
		require.NoError(err)
		sourceFileOne, err := ioutil.TempFile(sourceSubDirOne, "testCopyStepFile")
		require.NoError(err)

		rand.Seed(time.Now().UnixNano())
		sourceFileOneContent := make([]byte, 1048576)
		rand.Read(sourceFileOneContent)
		_, err = sourceFileOne.Write(sourceFileOneContent)
		require.NoError(err)
		defer sourceFileOne.Close()

		sourceFileOneRelPath, err := filepath.Rel(context.ContextDir, sourceFileOne.Name())
		require.NoError(err)

		target := "testCopyStepCommitOnNonCriticalPath_OneSourceFileToTargetFile/output"

		srcs := []string{sourceFileOneRelPath}

		// Copy to layer tar store.
		step := CopyStepFixture("", "", srcs, filepath.Join(workingDir, target), true)
		require.NoError(step.Execute(context, false))
		digestPairs, err := step.Commit(context)
		require.NoError(err)
		require.Len(digestPairs, 1)

		// Verify layer tar content.
		sha256 := digestPairs[0].GzipDescriptor.Digest.Hex()
		r, err := context.ImageStore.Layers.GetStoreFileReader(sha256)
		require.NoError(err)
		defer r.Close()
		gzipReader, err := tario.NewGzipReader(r)
		require.NoError(err)
		defer gzipReader.Close()
		gzipTarReader := tar.NewReader(gzipReader)
		for {
			header, err := gzipTarReader.Next()
			if err == io.EOF {
				break
			}
			require.NoError(err)

			name := header.Name
			switch header.Typeflag {
			case tar.TypeDir:
				continue
			case tar.TypeSymlink:
				continue
			case tar.TypeReg:
				require.Equal(strings.TrimLeft(filepath.Join(workingDir, target), "/"), name)
				b := make([]byte, len(sourceFileOneContent))
				_, err = gzipTarReader.Read(b)
				require.NoError(err)
				require.Equal(sourceFileOneContent, sourceFileOneContent)
			default:
				continue
			}
		}
	})
}
