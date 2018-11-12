package cache_test

import (
	"testing"

	"github.com/uber/makisu/lib/cache"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/registry"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestNoopCache(t *testing.T) {
	require := require.New(t)

	imageName, err := image.ParseName("repo:tag")
	require.NoError(err)

	cacheMgr := cache.New(nil, imageName, registry.NoopClientFixture())

	_, err = cacheMgr.PullCache("cacheid1")
	require.Equal(cache.ErrorLayerNotFound, errors.Cause(err))

	err = cacheMgr.PushCache(
		"cacheid2",
		&image.DigestPair{
			TarDigest:      image.Digest("sha256:test"),
			GzipDescriptor: image.Descriptor{Digest: image.Digest("sha256:testgzip")},
		},
	)
	require.NoError(err)
	err = cacheMgr.WaitForPush()
	require.NoError(err)
}

func TestMemKVStore(t *testing.T) {
	require := require.New(t)

	imageName, err := image.ParseName("registry.net/repo:tag")
	require.NoError(err)
	kvstore := cache.MemKVStore{}

	cacheMgr := cache.New(kvstore, imageName, registry.NoopClientFixture())

	_, err = cacheMgr.PullCache("cacheid1")
	require.Equal(cache.ErrorLayerNotFound, errors.Cause(err))

	err = cacheMgr.PushCache(
		"cacheid2",
		&image.DigestPair{
			TarDigest:      image.Digest("sha256:test"),
			GzipDescriptor: image.Descriptor{Digest: image.Digest("sha256:testgzip")},
		},
	)
	require.NoError(err)
	err = cacheMgr.WaitForPush()
	require.NoError(err)

	_, err = cacheMgr.PullCache("cacheid2")
	require.NoError(err)
}
