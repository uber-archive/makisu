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

package keyvalue

// MemStore implements Client interface. It stores cache key-value mappings
// in memory.
type MemStore map[string]string

// Get returns the value of a key previously set in memory.
func (m MemStore) Get(key string) (string, error) {
	return m[key], nil
}

// Put stores a key and its value in memory.
func (m MemStore) Put(key, value string) error {
	m[key] = value
	return nil
}

// Cleanup does nothing, but is implemented to comply with Client interface.
func (m MemStore) Cleanup() error { return nil }
