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

func TestLayerStoreCreateDownloadFile(t *testing.T) {
	require := require.New(t)

	root, err := ioutil.TempDir("/tmp", "makisu-test")
	require.NoError(err)
	store, err := NewImageStore(root)
	require.NoError(err)
	defer os.RemoveAll(root)

	repoName := "test_repo"
	repoName2 := "test_repo2"
	require.NoError(store.Layers.CreateDownloadFile(repoName, 1))

	_, err = store.Layers.GetDownloadOrCacheFileStat(repoName)
	require.NoError(err)
	_, err = store.Layers.GetDownloadFileReader(repoName)
	require.NoError(err)

	require.NoError(store.Layers.MoveDownloadFileToStore(repoName))
	_, err = store.Layers.GetStoreFileStat(repoName)
	require.NoError(err)
	_, err = store.Layers.GetStoreFileReader(repoName)
	require.NoError(err)
	require.NoError(store.Layers.LinkStoreFileTo(repoName, filepath.Join(root, "tmp")))
	require.NoError(store.Layers.LinkStoreFileFrom(repoName2, filepath.Join(root, "tmp")))

	require.NoError(store.Layers.DeleteStoreFile(repoName))
	require.NoError(store.Layers.DeleteStoreFile(repoName2))
}

func TestLayerTarStore(t *testing.T) {
	require := require.New(t)

	root, err := ioutil.TempDir("/tmp", "makisu-test")
	require.NoError(err)
	defer os.RemoveAll(root)
	store, err := NewImageStore(root)
	require.NoError(err)

	var waitGroup sync.WaitGroup
	for i := 0; i < 100; i++ {
		waitGroup.Add(1)

		go func(index int) {
			testFileName := fmt.Sprintf("test_%d", index)

			err := store.Layers.CreateDownloadFile(testFileName, 1)
			require.NoError(err)
			_, err = os.Stat(filepath.Join(root, layerTarDownloadDir, testFileName))
			require.NoError(err)

			err = store.Layers.MoveDownloadFileToStore(testFileName)
			require.NoError(err)
			_, err = os.Stat(filepath.Join(root, layerTarDownloadDir, testFileName))
			require.True(os.IsNotExist(err))
			_, err = os.Stat(filepath.Join(root, layerTarCacheDir, testFileName))
			require.NoError(err)

			_, err = store.Layers.GetStoreFileStat(testFileName)
			require.NoError(err)
			_, err = store.Layers.GetDownloadOrCacheFileStat(testFileName)
			require.NoError(err)
			err = store.Layers.LinkStoreFileTo(testFileName, filepath.Join(root, testFileName))
			require.NoError(err)

			err = store.Layers.DeleteStoreFile(testFileName)
			require.NoError(err)
			_, err = os.Stat(filepath.Join(root, layerTarDownloadDir, testFileName))
			require.True(os.IsNotExist(err))
			_, err = os.Stat(filepath.Join(root, layerTarCacheDir, testFileName))
			require.True(os.IsNotExist(err))

			waitGroup.Done()
		}(i)
	}

	waitGroup.Wait()
}
