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

package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManifestStoreCreateDownloadFile(t *testing.T) {
	require := require.New(t)

	root, err := ioutil.TempDir("/tmp", "makisu-test")
	require.NoError(err)
	store, err := NewImageStore(root)
	require.NoError(err)
	defer os.RemoveAll(root)

	repoName := "test_repo"
	repoName2 := "test_repo2"
	tagName := "test_tag"
	require.NoError(store.Manifests.CreateDownloadFile(repoName, tagName, 1))

	fileName := encodeRepoTag(repoName, tagName)
	_, err = os.Stat(filepath.Join(root, manifestDownloadDir, fileName))
	require.NoError(err)
	_, err = store.Manifests.GetDownloadOrCacheFileStat(repoName, tagName)
	require.NoError(err)

	require.NoError(store.Manifests.MoveDownloadFileToStore(repoName, tagName))
	_, err = store.Manifests.GetStoreFileStat(repoName, tagName)
	require.NoError(err)
	_, err = store.Manifests.GetStoreFileReader(repoName, tagName)
	require.NoError(err)
	require.NoError(store.Manifests.LinkStoreFileTo(repoName, tagName, filepath.Join(root, "tmpfile")))
	require.NoError(store.Manifests.LinkStoreFileFrom(repoName2, tagName, filepath.Join(root, "tmpfile")))

	require.NoError(store.Manifests.DeleteStoreFile(repoName, tagName))
	require.NoError(store.Manifests.DeleteStoreFile(repoName2, tagName))
}

func TestManifestStore(t *testing.T) {
	require := require.New(t)

	root, err := ioutil.TempDir("/tmp", "makisu-test")
	require.NoError(err)
	defer os.RemoveAll(root)
	store, err := NewImageStore(root)
	require.NoError(err)

	var waitGroup sync.WaitGroup

	for i := 0; i < 10; i++ {
		waitGroup.Add(1)

		go func(index int) {
			repoName := ("test_repo")
			tagName := fmt.Sprintf("test_%d", index)
			fileName := encodeRepoTag(repoName, tagName)
			parsedRepoName, parsedTagName, err := decodeRepoTag(fileName)
			require.NoError(err)
			require.Equal(repoName, parsedRepoName)
			require.Equal(tagName, parsedTagName)

			err = store.Manifests.CreateDownloadFile(repoName, tagName, 1)
			require.NoError(err)
			_, err = os.Stat(filepath.Join(root, manifestDownloadDir, fileName))
			require.NoError(err)

			err = store.Manifests.MoveDownloadFileToStore(repoName, tagName)
			require.NoError(err)
			_, err = os.Stat(filepath.Join(root, manifestDownloadDir, fileName))
			require.True(os.IsNotExist(err))
			_, err = os.Stat(filepath.Join(root, manifestCacheDir, fileName))
			require.NoError(err)

			err = store.Manifests.DeleteStoreFile(repoName, tagName)
			require.NoError(err)
			_, err = os.Stat(filepath.Join(root, manifestDownloadDir, fileName))
			require.True(os.IsNotExist(err))
			_, err = os.Stat(filepath.Join(root, manifestCacheDir, fileName))
			require.True(os.IsNotExist(err))

			waitGroup.Done()
		}(i)
	}

	waitGroup.Wait()
}
