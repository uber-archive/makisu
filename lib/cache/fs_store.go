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
	"encoding/json"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

type cacheEntry struct {
	layerSHA  string
	timestamp int64
}

type fsStore struct {
	sync.Mutex

	fullpath   string
	sandboxDir string
	ttlsec     int64

	entries map[string]cacheEntry
}

// NewFSStore returns a KVStore backed by the local filesystem.
// Entries are stored in json format.
// TODO: enforce capacity.
func NewFSStore(fullpath string, sandboxDir string, ttlsec int64) (KVStore, error) {
	s := &fsStore{
		fullpath:   fullpath,
		sandboxDir: sandboxDir,
		ttlsec:     ttlsec,
		entries:    make(map[string]cacheEntry),
	}

	contents, err := ioutil.ReadFile(fullpath)
	if os.IsNotExist(err) {
		return s, nil
	} else if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(contents, &s.entries); err != nil {
		if err := os.Remove(fullpath); err != nil {
			return nil, err
		}
		return s, nil
	}

	// Remove entries that's older than TTL.
	for key, entry := range s.entries {
		if time.Now().Unix()-entry.timestamp > s.ttlsec {
			// Cache expired.
			delete(s.entries, key)
		}
	}

	return s, nil
}

func (s *fsStore) Get(key string) (string, error) {
	entry, ok := s.entries[key]
	if !ok {
		return "", nil
	}
	// Update timestamp.
	entry.timestamp = time.Now().Unix()

	return entry.layerSHA, nil
}

func (s *fsStore) Put(key, value string) error {
	entry := cacheEntry{
		layerSHA:  value,
		timestamp: time.Now().Unix(),
	}

	s.entries[key] = entry

	content, err := json.Marshal(s.entries)
	if err != nil {
		return err
	}

	tempFile, err := ioutil.TempFile(s.sandboxDir, "cache")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name())

	if err := ioutil.WriteFile(tempFile.Name(), content, 0755); err != nil {
		return err
	}
	if err := os.Rename(tempFile.Name(), s.fullpath); err != nil {
		return err
	}

	return nil
}

func (s *fsStore) Cleanup() error {
	s.entries = make(map[string]cacheEntry)

	return os.Remove(s.fullpath)
}
