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

// KVStore is the interface that the CacheManager relies on to make the mapping between cacheID
// and image name.
// The Get function returns an empty string and no error if the key was not found in the store.
// Cleanup closes potential connections to the store.
type KVStore interface {
	Get(string) (string, error)
	Put(string, string) error
	Cleanup() error
}

// MemKVStore implements the KVStore interface. It stores the key-value mappings in memory.
type MemKVStore map[string]string

// Get returns the value of a key previously set in memory.
func (m MemKVStore) Get(key string) (string, error) {
	return m[key], nil
}

// Put stores a key and its value in memory.
func (m MemKVStore) Put(key, value string) error {
	m[key] = value
	return nil
}

// Cleanup does nothing, but is implemented to comply with the KVStore interface.
func (m MemKVStore) Cleanup() error { return nil }
