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

package cache_test

import (
	"sync"
	"testing"

	"github.com/uber/makisu/lib/cache"
	"github.com/uber/makisu/lib/cache/keyvalue"
	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/registry"
	mockregistry "github.com/uber/makisu/mocks/lib/registry"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestNoopCache(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	cacheMgr := cache.New(ctx.ImageStore, nil, registry.NoopClientFixture())

	_, err := cacheMgr.PullCache("cacheid1")
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

func TestCachePushAndPull(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	kvStore := keyvalue.MockStore{}
	cacheMgr := cache.New(ctx.ImageStore, kvStore, registry.NoopClientFixture())

	_, err := cacheMgr.PullCache("cacheid1")
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

func TestCachePullWithOngoingPushing(t *testing.T) {
	require := require.New(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	var wg sync.WaitGroup
	wg.Add(1)

	mockClient := mockregistry.NewMockClient(ctrl)
	mockClient.EXPECT().PullLayer(image.Digest("sha256:testgzip")).Return(nil, nil)
	mockClient.EXPECT().PushLayer(image.Digest("sha256:testgzip")).
		Do(func(d image.Digest) {
			wg.Done()
			// Push doesn't return until the end of the test.
			select {}
		}).Return(nil)

	kvStore := keyvalue.MockStore{}
	cacheMgr := cache.New(ctx.ImageStore, kvStore, mockClient)

	_, err := cacheMgr.PullCache("cacheid1")
	require.Equal(cache.ErrorLayerNotFound, errors.Cause(err))

	go func() {
		err = cacheMgr.PushCache(
			"cacheid2",
			&image.DigestPair{
				TarDigest:      image.Digest("sha256:test"),
				GzipDescriptor: image.Descriptor{Digest: image.Digest("sha256:testgzip")},
			},
		)
	}()

	// Wait until layer is being pushed to make sure memKVStore is populated.
	wg.Wait()
	_, err = cacheMgr.PullCache("cacheid2")
	require.NoError(err)
}
