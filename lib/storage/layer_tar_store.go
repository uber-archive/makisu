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
	"path"
	"time"

	"github.com/andres-erbsen/clock"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/storage/base"
	"github.com/uber/makisu/lib/storage/metadata"
	"github.com/uber/makisu/lib/utils"
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
	err := os.MkdirAll(downloadDir, 0755)
	utils.Must(err == nil, "Failed to create layer download dir %s: %s", downloadDir, err)

	// We do not want to remove existing files in store directory during restart.
	err = os.MkdirAll(cacheDir, 0755)
	utils.Must(err == nil, "Failed to create layer storage dir %s: %s", cacheDir, err)

	backend := base.NewLRUFileStore(layerLRUSize, clock.New())
	downloadState := base.NewFileState(downloadDir)
	cacheState := base.NewFileState(cacheDir)

	// Reload all existing data
	files, err := ioutil.ReadDir(cacheDir)
	utils.Must(err == nil, "Failed to scan layer storage dir %s: %s", cacheDir, err)
	for _, f := range files {
		_, err := backend.NewFileOp().AcceptState(cacheState).GetFileStat(f.Name())
		utils.Must(err == nil, "Failed to load layer storage dir %s: %s", cacheDir, err)
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

// cleanup scans the store for idle or expired files.
// Also returns the total disk usage.
func (s *LayerTarStore) cleanup(
	op base.FileOp, tti time.Duration, ttl time.Duration) (usage int64, err error) {

	names, err := s.backend.NewFileOp().AcceptState(s.cacheState).ListNames()
	if err != nil {
		return 0, fmt.Errorf("list names: %s", err)
	}
	for _, name := range names {
		info, err := op.GetFileStat(name)
		if err != nil {
			log.With("name", name).Errorf("Error getting file stat: %s", err)
			continue
		}
		if ready, err := s.readyForDeletion(op, name, info, tti, ttl); err != nil {
			log.With("name", name).Errorf("Error checking if file expired: %s", err)
		} else if ready {
			if err := op.DeleteFile(name); err != nil && err != base.ErrFilePersisted {
				log.With("name", name).Errorf("Error deleting expired file: %s", err)
			}
		}
		usage += info.Size()
	}
	return usage, nil
}

func (s *LayerTarStore) readyForDeletion(
	op base.FileOp,
	name string,
	info os.FileInfo,
	tti time.Duration,
	ttl time.Duration) (bool, error) {

	if ttl > 0 && s.clk.Now().Sub(info.ModTime()) > ttl {
		return true, nil
	}

	var lat metadata.LastAccessTime
	if err := op.GetFileMetadata(name, &lat); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("get file lat: %s", err)
	}
	return s.clk.Now().Sub(lat.Time) > tti, nil
}
