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
