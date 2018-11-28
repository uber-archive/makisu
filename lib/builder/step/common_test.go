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
	"os"
	"strings"
	"testing"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/tario"

	"github.com/stretchr/testify/require"
)

func readGzippedTar(t *testing.T, f io.Reader) map[string]os.FileInfo {
	require := require.New(t)

	gzipReader, err := tario.NewGzipReader(f)
	require.NoError(err)
	defer gzipReader.Close()

	files := make(map[string]os.FileInfo)
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		require.NoError(err)

		files["/"+strings.TrimLeft(header.Name, "/")] = header.FileInfo()
	}

	return files
}

func TestTarAndGzipDiffsEmpty(t *testing.T) {
	require := require.New(t)

	context, cleanup := context.BuildContextFixture()
	defer cleanup()

	_, _, name, err := tarAndGzipDiffs(context, func(*tar.Writer) error { return nil })
	require.NoError(err)

	f, err := os.Open(name)
	require.NoError(err)
	defer f.Close()

	files := readGzippedTar(t, f)
	require.Empty(files)
}

func TestTarAndGzipDiffsAddedFile(t *testing.T) {
	require := require.New(t)

	context, cleanup := context.BuildContextFixture()
	defer cleanup()

	f, err := ioutil.TempFile(context.RootDir, "testTarAndGzipDiffs")
	filename := f.Name()
	require.NoError(err)
	defer f.Close()

	_, _, tmpName, err := tarAndGzipDiffs(context, context.MemFS.AddLayerByScan)
	require.NoError(err)
	defer os.Remove(tmpName)

	f, err = os.Open(tmpName)
	require.NoError(err)
	defer f.Close()

	files := readGzippedTar(t, f)
	require.Equal(1, len(files))
	require.Contains(files, strings.TrimPrefix(filename, context.RootDir))
}

func TestCommitDiffs(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	tests := []struct {
		runStep          *RunStep
		verifyGzippedTar func(io.Reader)
	}{
		{
			NewRunStep("", "touch file1 && touch file2", true),
			func(f io.Reader) {
				files := readGzippedTar(t, f)
				require.Equal(2, len(files))
				require.Contains(files, "/file1")
				require.Contains(files, "/file2")
			},
		},
		{
			NewRunStep("", "mkdir dir1 && rm file1", true),
			func(f io.Reader) {
				files := readGzippedTar(t, f)
				require.Equal(2, len(files))
				require.Contains(files, "/dir1/")
				require.Contains(files, "/.wh.file1")
			},
		},
		{
			NewRunStep("", "rm -rf dir1", true),
			func(f io.Reader) {
				files := readGzippedTar(t, f)
				require.Equal(1, len(files))
				require.Contains(files, "/.wh.dir1")
			},
		},
		{
			NewRunStep("", "ls ./", true),
			func(f io.Reader) {
				// Verify no files were tarred, since the command doesn't write to or create any files.
				files := readGzippedTar(t, f)
				require.Equal(0, len(files))
			},
		},
	}

	for _, test := range tests {
		require.NoError(test.runStep.ApplyCtxAndConfig(ctx, nil))
		require.NoError(test.runStep.Execute(ctx, true))
		digestPairs, err := test.runStep.Commit(ctx)
		require.NoError(err)
		require.Len(digestPairs, 1)

		f, err := ctx.ImageStore.Layers.GetStoreFileReader(digestPairs[0].GzipDescriptor.Digest.Hex())
		require.NoError(err)
		defer f.Close()
		test.verifyGzippedTar(f)
	}
}
