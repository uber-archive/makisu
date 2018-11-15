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

package base

import (
	"sync"
)

// FileMap is a thread-safe name -> FileEntry map.
type FileMap interface {
	Contains(name string) bool
	LoadOrStore(name string, entry FileEntry, f func(string, FileEntry) error) (FileEntry, bool)
	LoadForWrite(name string, f func(string, FileEntry)) bool
	LoadForRead(name string, f func(string, FileEntry)) bool
	LoadForPeek(name string, f func(string, FileEntry)) bool
	Delete(name string, f func(string, FileEntry) error) bool
}

var _ FileMap = (*simpleFileMap)(nil)

type fileEntryWithRWLock struct {
	sync.RWMutex

	fe FileEntry
}

// simpleFileMap is a two-level locking map which synchronizes access to the
// map in addition to synchronizing access to the values within the map. Useful
// for mutating values in-place.
//
// The zero Map is valid and empty.
type simpleFileMap struct {
	m sync.Map
}

// NewSimpleFileMap inits a new simpleFileMap object.
func NewSimpleFileMap() FileMap {
	return &simpleFileMap{}
}

// Contains returns true if the given key is stored in the map.
func (fm *simpleFileMap) Contains(name string) bool {
	_, loaded := fm.m.Load(name)

	return loaded
}

// LoadOrStore tries to stores the given key / value pair into the map.
// If entry was successfully put into the map, execute f under the protection of Lock.
// Returns existing oject and true if the name is already present.
func (fm *simpleFileMap) LoadOrStore(
	name string, entry FileEntry, f func(string, FileEntry) error) (FileEntry, bool) {
	// Grab entry lock first, in case other goroutines get the lock between LoadOrStore() and f().
	e := &fileEntryWithRWLock{
		fe: entry,
	}
	e.Lock()
	defer e.Unlock()

	if actual, loaded := fm.m.LoadOrStore(name, e); loaded {
		return actual.(*fileEntryWithRWLock).fe, true
	}

	if err := f(name, e.fe); err != nil {
		// Remove from map while the entry lock is still being held
		fm.m.Delete(name)
		return nil, false
	}
	return entry, false
}

// LoadForWrite looks up the value of key k and executes f under the protection of Lock.
// While f executes, it is guaranteed that k will not be deleted from the map.
// Returns false if k was not found.
func (fm *simpleFileMap) LoadForWrite(name string, f func(string, FileEntry)) bool {
	v, ok := fm.m.Load(name)
	if !ok {
		return false
	}

	e := v.(*fileEntryWithRWLock)
	e.Lock()
	defer e.Unlock()

	// Now that we have the entry lock, make sure k was not deleted or overwritten.
	if nv, ok := fm.m.Load(name); !ok {
		return false
	} else if nv != v {
		return false
	}

	f(name, e.fe)

	return true
}

// LoadForRead looks up the value of key k and executes f under the protection of RLock.
// While f executes, it is guaranteed that k will not be deleted from the map.
// Returns false if k was not found.
func (fm *simpleFileMap) LoadForRead(name string, f func(string, FileEntry)) bool {
	v, ok := fm.m.Load(name)
	if !ok {
		return false
	}

	e := v.(*fileEntryWithRWLock)
	e.RLock()
	defer e.RUnlock()

	// Now that we have the entry lock, make sure k was not deleted or overwritten.
	if nv, ok := fm.m.Load(name); !ok {
		return false
	} else if nv != v {
		return false
	}

	f(name, e.fe)

	return true
}

// LoadForPeek is the same a LoadForRead in this implementation.
func (fm *simpleFileMap) LoadForPeek(name string, f func(string, FileEntry)) bool {
	return fm.LoadForRead(name, f)
}

// Delete deletes the given key from the Map.
// It also executes f under the protection of Lock.
// If f returns false, abort before key deletion.
func (fm *simpleFileMap) Delete(name string, f func(string, FileEntry) error) bool {
	v, ok := fm.m.Load(name)
	if !ok {
		return false
	}

	e := v.(*fileEntryWithRWLock)
	e.Lock()
	defer e.Unlock()

	// Now that we have the entry lock, make sure k was not deleted or overwritten.
	if nv, ok := fm.m.Load(name); !ok {
		return false
	} else if nv != v {
		return false
	}

	if err := f(name, e.fe); err != nil {
		return false
	}

	fm.m.Delete(name)
	return true
}
