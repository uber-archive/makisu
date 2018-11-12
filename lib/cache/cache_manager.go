package cache

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/registry"
	"github.com/uber/makisu/lib/utils"

	"github.com/pkg/errors"
)

const _cachePrefix = "ubuild_engine_cache_"

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
	cacheIDStore   KVStore
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
func New(cacheIDStore KVStore, target image.Name, registryClient registry.Client) Manager {
	if cacheIDStore == nil {
		log.Infof("No registry or KV store provided, using noop cache manager")
		return noopCacheManager{}
	}
	return &registryCacheManager{
		cacheIDStore:   cacheIDStore,
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
		entry, err = manager.cacheIDStore.Get(_cachePrefix + cacheID)
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

	tarDigest, gzipDigest, err := parseEntry(entry)
	if err != nil {
		return nil, errors.Wrapf(ErrorLayerNotFound, "parse entry %s", entry)
	}

	// Pull layer from docker registry.
	info, err := manager.registryClient.PullLayer(gzipDigest)
	if err != nil {
		return nil, fmt.Errorf("pull layer %s: %s", entry, err)
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
	manager.wg.Add(1)

	go func() {
		defer manager.wg.Done()

		manager.Lock()
		defer manager.Unlock()

		if err := manager.registryClient.PushLayer(digestPair.GzipDescriptor.Digest); err != nil {
			manager.pushErrors.Add(fmt.Errorf("push layer %s: %s", digestPair.GzipDescriptor.Digest, err))
			return
		}

		entry := createEntry(digestPair.TarDigest, digestPair.GzipDescriptor.Digest)
		if err := manager.cacheIDStore.Put(_cachePrefix+cacheID, entry); err != nil {
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

func createEntry(tarDigest, gzipDigest image.Digest) string {
	return fmt.Sprintf("%s,%s", tarDigest.Hex(), gzipDigest.Hex())
}
