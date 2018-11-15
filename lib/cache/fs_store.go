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
