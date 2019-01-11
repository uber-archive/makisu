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
	"io/ioutil"
	"os"
	"path"

	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/storage/base"

	"github.com/andres-erbsen/clock"
)

// layerTarState implements FileState interface, which is needed by FileStore.
type layerTarState int

const (
	layerTarDownloadDir = "layer_tar/download"
	layerTarCacheDir    = "layer_tar/cache"
)
const layerLRUSize = 256

// LayerTarStore manages layer tar files on local disk.
type LayerTarStore struct {
	backend       base.FileStore
	downloadState base.FileState
	cacheState    base.FileState
}

// NewLayerTarStore initializes and returns a new LayerTarStore object.
func NewLayerTarStore(rootdir string) (*LayerTarStore, error) {
	// Init all directories.
	downloadDir := path.Join(rootdir, layerTarDownloadDir)
	cacheDir := path.Join(rootdir, layerTarCacheDir)

	// Remove and recreate download dir.
	os.RemoveAll(downloadDir)
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		log.Fatalf("Failed to create layer download dir %s: %s", downloadDir, err)
	}

	// We do not want to remove existing files in store directory during restart.
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		log.Fatalf("Failed to create layer cache dir %s: %s", cacheDir, err)
	}

	backend := base.NewLRUFileStore(layerLRUSize, clock.New())
	downloadState := base.NewFileState(downloadDir)
	cacheState := base.NewFileState(cacheDir)

	// Reload all existing data
	files, err := ioutil.ReadDir(cacheDir)
	if err != nil {
		log.Fatalf("Failed to scan layer cache dir %s: %s", cacheDir, err)
	}
	for _, f := range files {
		if _, err := backend.NewFileOp().AcceptState(cacheState).GetFileStat(f.Name()); err != nil {
			// Probably caused by an empty directory. Try detele.
			log.Warnf("Failed to load cached manifest: %s", err)
			if err := backend.NewFileOp().AcceptState(cacheState).DeleteFile(f.Name()); err != nil {
				log.Warnf("Failed to cleanup cached manifest: %s", err)
			}
		}
	}

	return &LayerTarStore{
		backend:       backend,
		downloadState: downloadState,
		cacheState:    cacheState,
	}, nil
}

// CreateDownloadFile creates an empty file in download directory with specified size.
func (s *LayerTarStore) CreateDownloadFile(fileName string, len int64) error {
	return s.backend.NewFileOp().AcceptState(s.downloadState).CreateFile(
		fileName, s.downloadState, len)
}

// GetDownloadFileReader returns a FileReader for a file in download directory.
func (s *LayerTarStore) GetDownloadFileReader(fileName string) (base.FileReader, error) {
	return s.backend.NewFileOp().AcceptState(s.downloadState).GetFileReader(fileName)
}

// GetDownloadFileReadWriter returns a FileReadWriter for a file in download directory.
func (s *LayerTarStore) GetDownloadFileReadWriter(fileName string) (base.FileReadWriter, error) {
	return s.backend.NewFileOp().AcceptState(s.downloadState).GetFileReadWriter(fileName)
}

// MoveDownloadFileToStore moves a file from store directory to cache directory.
func (s *LayerTarStore) MoveDownloadFileToStore(fileName string) error {
	return s.backend.NewFileOp().AcceptState(s.downloadState).MoveFile(fileName, s.cacheState)
}

// LinkStoreFileFrom create a hardlink in store from given source path.
func (s *LayerTarStore) LinkStoreFileFrom(fileName, src string) error {
	return s.backend.NewFileOp().AcceptState(s.cacheState).MoveFileFrom(fileName, s.cacheState, src)
}

// GetStoreFileReader returns a FileReader for a file in store directory.
func (s *LayerTarStore) GetStoreFileReader(fileName string) (base.FileReader, error) {
	return s.backend.NewFileOp().AcceptState(s.cacheState).GetFileReader(fileName)
}

// GetDownloadOrCacheFileStat returns os.FileInfo for a file in download or cache directory.
func (s *LayerTarStore) GetDownloadOrCacheFileStat(fileName string) (os.FileInfo, error) {
	return s.backend.NewFileOp().AcceptState(s.downloadState).AcceptState(s.cacheState).GetFileStat(
		fileName)
}

// GetStoreFileStat returns FileInfo of the specified file.
func (s *LayerTarStore) GetStoreFileStat(fileName string) (os.FileInfo, error) {
	return s.backend.NewFileOp().AcceptState(s.cacheState).GetFileStat(fileName)
}

// DeleteStoreFile deletes a file from store directory.
func (s *LayerTarStore) DeleteStoreFile(fileName string) error {
	return s.backend.NewFileOp().AcceptState(s.cacheState).DeleteFile(fileName)
}

// LinkStoreFileTo hardlinks file from store to target
func (s *LayerTarStore) LinkStoreFileTo(fileName, target string) error {
	return s.backend.NewFileOp().AcceptState(s.cacheState).LinkFileTo(fileName, target)
}
