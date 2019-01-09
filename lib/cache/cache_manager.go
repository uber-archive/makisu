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
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/uber/makisu/lib/cache/keyvalue"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/registry"
	"github.com/uber/makisu/lib/storage"
	"github.com/uber/makisu/lib/utils"

	"github.com/pkg/errors"
)

const _cachePrefix = "makisu_builder_cache_"
const _cacheEmptyEntry = "MAKISU_CACHE_EMPTY"

// Manager is the interface through which we interact with the cacheID -> image layer mapping.
type Manager interface {
	PullCache(cacheID string) (*image.DigestPair, error)
	PushCache(cacheID string, digestPair *image.DigestPair) error
	WaitForPush() error
}

// noopCacheManager is an implementation of the cache.Manager interface.
// The PullCache implementation returns errors.Wrap(ErrorLayerNotFound), and the other methods are
// noops.
type noopCacheManager struct{}

// NewNoopCacheManager returns a Manager that does nothing.
func NewNoopCacheManager() Manager { return noopCacheManager{} }

func (manager noopCacheManager) PullCache(cacheID string) (*image.DigestPair, error) {
	return nil, errors.Wrapf(ErrorLayerNotFound, "Unable to find layer %s in Noop cache", cacheID)
}

func (manager noopCacheManager) PushCache(cacheID string, digestPair *image.DigestPair) error {
	return nil
}

func (manager noopCacheManager) WaitForPush() error {
	return nil
}

// registryCacheManager uses a docker registry as cache layer storage.
// It needs an additional key-value store for cache key/layer name lookup.
// It implements CacheManager interface.
type registryCacheManager struct {
	imageStore     *storage.ImageStore
	kvStore        keyvalue.Store
	registryClient registry.Client

	sync.Mutex
	wg         sync.WaitGroup
	pushErrors utils.MultiErrors
}

var (
	// ErrorLayerNotFound is the error returned by Lookup when the layer
	// requested was not found in the registry.
	ErrorLayerNotFound = errors.Errorf("layer not found in cache")
)

// New returns a new cache manager that interacts with the registry
// passed in as well as the local filesystem through the image store.
// By default the registry field is left blank.
func New(
	imageStore *storage.ImageStore, kvStore keyvalue.Store,
	registryClient registry.Client) Manager {

	if imageStore == nil || kvStore == nil {
		log.Infof("No image store or KV store provided, using noop cache manager")
		return noopCacheManager{}
	}
	return &registryCacheManager{
		imageStore:     imageStore,
		kvStore:        kvStore,
		registryClient: registryClient,
	}
}

// PullCache tries to fetch the layer corresponding to the cache ID.
// If the layer is not found, it returns ErrorLayerNotFound.
// This function is blocking
func (manager *registryCacheManager) PullCache(cacheID string) (*image.DigestPair, error) {
	manager.Lock()
	defer manager.Unlock()

	var entry string
	var err error
	for i := 0; ; i++ {
		entry, err = manager.kvStore.Get(_cachePrefix + cacheID)
		if err == nil && entry != "" {
			break
		} else if entry == "" {
			return nil, errors.Wrapf(ErrorLayerNotFound, "find layer %s", cacheID)
		} else {
			if i >= 2 {
				return nil, fmt.Errorf("query cache id %s: %s", cacheID, err)
			}
			log.Info("Retrying query for cacheID %s", cacheID)
			time.Sleep(time.Second)
		}
	}
	log.Infof("Found mapping in cacheID KVStore: %s => %s", cacheID, entry)

	if entry == _cacheEmptyEntry {
		return nil, nil
	}

	tarDigest, gzipDigest, err := parseEntry(entry)
	if err != nil {
		return nil, errors.Wrapf(ErrorLayerNotFound, "parse entry %s", entry)
	}

	// Check if layer is already on disk.
	info, err := manager.imageStore.Layers.GetStoreFileStat(gzipDigest.Hex())
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("stat layer %s: %s", entry, err)
	} else if os.IsNotExist(err) {
		if manager.registryClient == nil {
			return nil, fmt.Errorf("registry client not configured to pull cache")
		}
		// Pull layer from docker registry.
		info, err = manager.registryClient.PullLayer(gzipDigest)
		if err != nil {
			return nil, fmt.Errorf("pull layer %s: %s", entry, err)
		}
	}

	// Info might be nil if the registry client is a test fixture.
	var size int64
	if info != nil {
		size = info.Size()
	}
	return &image.DigestPair{
		TarDigest: tarDigest,
		GzipDescriptor: image.Descriptor{
			MediaType: image.MediaTypeLayer,
			Size:      size,
			Digest:    gzipDigest,
		},
	}, nil
}

// PushCache tries to push an image layer asynchronously.
func (manager *registryCacheManager) PushCache(cacheID string, digestPair *image.DigestPair) error {
	if manager.registryClient == nil {
		manager.pushErrors.Add(fmt.Errorf("registry client not configured to push cache"))
		return nil
	}

	manager.wg.Add(1)

	go func() {
		defer manager.wg.Done()

		manager.Lock()
		defer manager.Unlock()

		if digestPair != nil {
			if err := manager.registryClient.PushLayer(digestPair.GzipDescriptor.Digest); err != nil {
				manager.pushErrors.Add(fmt.Errorf("push layer %s: %s", digestPair.GzipDescriptor.Digest, err))
				return
			}
		}

		entry := createEntry(digestPair)
		if err := manager.kvStore.Put(_cachePrefix+cacheID, entry); err != nil {
			manager.pushErrors.Add(fmt.Errorf("store tag mapping (%s,%s): %s", cacheID, entry, err))
			return
		}

		log.Infof("Stored cacheID mapping to KVStore: %s => %s", cacheID, entry)
	}()

	return nil
}

// WaitForPush blocks until all cache pushes are done or timeout.
func (manager *registryCacheManager) WaitForPush() error {
	c := make(chan struct{})
	go func() {
		defer close(c)
		manager.wg.Wait()
	}()
	select {
	case <-c:
		return manager.pushErrors.Collect()
	case <-time.After(time.Minute * 10):
		return fmt.Errorf("timeout waiting for push")
	}
}

func parseEntry(entry string) (image.Digest, image.Digest, error) {
	if strings.Index(entry, ",") == -1 {
		return image.NewEmptyDigest(), image.NewEmptyDigest(), errors.Errorf("parse redis entry: %s", entry)
	}
	split := strings.SplitN(entry, ",", 2)
	return image.Digest("sha256:" + split[0]), image.Digest("sha256:" + split[1]), nil
}

func createEntry(pair *image.DigestPair) string {
	if pair == nil {
		return _cacheEmptyEntry
	}
	return fmt.Sprintf("%s,%s", pair.TarDigest.Hex(), pair.GzipDescriptor.Digest.Hex())
}
