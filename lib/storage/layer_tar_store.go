package storage

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/andres-erbsen/clock"
	"github.com/uber/makisu/lib/storage/base"
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
