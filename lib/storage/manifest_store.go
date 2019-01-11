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
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/storage/base"

	"github.com/andres-erbsen/clock"
)

// manifestState implements FileState interface, which is needed by FileStore.
type manifestState int

const (
	manifestDownloadDir = "manifest/download"
	manifestCacheDir    = "manifest/cache"
)
const manifestLRUSize = 16

// ManifestStore manages image manifest files on local disk.
// It uses repo/tag as file name, distribution manifest as content.
// Notice manifest is not the same as image config.
type ManifestStore struct {
	backend       base.FileStore
	downloadState base.FileState
	cacheState    base.FileState
}

// NewManifestStore initializes and returns a new ManifestStore object.
func NewManifestStore(rootdir string) (*ManifestStore, error) {
	// Init all directories.
	downloadDir := path.Join(rootdir, manifestDownloadDir)
	cacheDir := path.Join(rootdir, manifestCacheDir)

	// Remove and recreate download dir.
	os.RemoveAll(downloadDir)
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		log.Fatalf("Failed to create manifest download dir %s: %s", downloadDir, err)
	}

	// We do not want to remove existing files in store directory during restart.
	// TODO: we could have dangling manifests
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		log.Fatalf("Failed to create manifest cache dir %s: %s", cacheDir, err)
	}

	backend := base.NewLRUFileStore(manifestLRUSize, clock.New())
	downloadState := base.NewFileState(downloadDir)
	cacheState := base.NewFileState(cacheDir)

	// Reload all existing data
	files, err := ioutil.ReadDir(cacheDir)
	if err != nil {
		log.Fatalf("Failed to scan manifest cache dir %s: %s", cacheDir, err)
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

	return &ManifestStore{
		backend:       backend,
		downloadState: downloadState,
		cacheState:    cacheState,
	}, nil
}

func encodeRepoTag(repo, tag string) string {
	unencoded := strings.Join([]string{repo, tag}, "/")
	return base64.StdEncoding.EncodeToString([]byte(unencoded))
}

func decodeRepoTag(fileName string) (string, string, error) {
	decoded, err := base64.StdEncoding.DecodeString(fileName)
	if err != nil {
		return "", "", err
	}
	parts := regexp.MustCompile(`^(.+)\/(\w+)$`).FindStringSubmatch(string(decoded))
	if parts == nil || len(parts) != 3 {
		return "", "", fmt.Errorf("Failed to parse repo/tag from file name")
	}
	return parts[1], parts[2], nil
}

// CreateDownloadFile creates an empty file in download directory with specified size.
func (s *ManifestStore) CreateDownloadFile(repo, tag string, len int64) error {
	fileName := encodeRepoTag(repo, tag)
	return s.backend.NewFileOp().AcceptState(s.downloadState).CreateFile(
		fileName, s.downloadState, len)
}

// GetDownloadFileReadWriter returns a FileReadWriter for a file in download directory.
func (s *ManifestStore) GetDownloadFileReadWriter(repo, tag string) (base.FileReadWriter, error) {
	fileName := encodeRepoTag(repo, tag)
	return s.backend.NewFileOp().AcceptState(s.downloadState).GetFileReadWriter(fileName)
}

// MoveDownloadFileToStore moves a file from store directory to cache directory.
func (s *ManifestStore) MoveDownloadFileToStore(repo, tag string) error {
	fileName := encodeRepoTag(repo, tag)
	return s.backend.NewFileOp().AcceptState(s.downloadState).MoveFile(fileName, s.cacheState)
}

// LinkStoreFileFrom create a hardlink in store from given source path.
func (s *ManifestStore) LinkStoreFileFrom(repo, tag, src string) error {
	fileName := encodeRepoTag(repo, tag)
	return s.backend.NewFileOp().AcceptState(s.cacheState).MoveFileFrom(fileName, s.cacheState, src)
}

// GetStoreFileReader returns a FileReader for a file in store directory.
func (s *ManifestStore) GetStoreFileReader(repo, tag string) (base.FileReader, error) {
	fileName := encodeRepoTag(repo, tag)
	return s.backend.NewFileOp().AcceptState(s.cacheState).GetFileReader(fileName)
}

// GetDownloadOrCacheFileStat returns os.FileInfo for a file in download or cache directory.
func (s *ManifestStore) GetDownloadOrCacheFileStat(repo, tag string) (os.FileInfo, error) {
	fileName := encodeRepoTag(repo, tag)
	return s.backend.NewFileOp().AcceptState(s.downloadState).AcceptState(s.cacheState).GetFileStat(
		fileName)
}

// GetStoreFileStat returns os.FileInfo for a file in store directory.
func (s *ManifestStore) GetStoreFileStat(repo, tag string) (os.FileInfo, error) {
	fileName := encodeRepoTag(repo, tag)
	return s.backend.NewFileOp().AcceptState(s.cacheState).GetFileStat(fileName)
}

// DeleteStoreFile deletes a file from store directory.
// TODO: deref all layers.
func (s *ManifestStore) DeleteStoreFile(repo, tag string) error {
	fileName := encodeRepoTag(repo, tag)
	return s.backend.NewFileOp().AcceptState(s.cacheState).DeleteFile(fileName)
}

// LinkStoreFileTo hardlinks file from store to target
func (s *ManifestStore) LinkStoreFileTo(repo, tag, target string) error {
	fileName := encodeRepoTag(repo, tag)
	return s.backend.NewFileOp().AcceptState(s.cacheState).LinkFileTo(fileName, target)
}
