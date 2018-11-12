package cache

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

type fsStore struct {
	sync.Mutex

	root string
}

// NewFSStore returns a KVStore backed by the local filesystem. Each key stored
// will correspond to a file on disk and its value is the contents of that key.
func NewFSStore(root string) KVStore {
	return &fsStore{
		root: root,
	}
}

func (store *fsStore) Get(key string) (string, error) {
	path := filepath.Join(store.root, key)
	contents, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	return string(contents), nil
}

func (store *fsStore) Put(key, value string) error {
	path := filepath.Join(store.root, key)
	return ioutil.WriteFile(path, []byte(value), 0677)
}

func (store *fsStore) Cleanup() error { return nil }
