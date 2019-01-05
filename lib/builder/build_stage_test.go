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

package builder

import (
	"testing"

	"github.com/uber/makisu/lib/cache"
	"github.com/uber/makisu/lib/cache/keyvalue"
	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/parser/dockerfile"
	"github.com/uber/makisu/lib/registry"

	"github.com/stretchr/testify/require"
)

var _testDigestPair = &image.DigestPair{
	TarDigest:      image.Digest("sha256:test"),
	GzipDescriptor: image.Descriptor{Digest: image.Digest("sha256:testgzip")},
}

func TestPullCacheLayers(t *testing.T) {
	testCases := []struct {
		name             string
		stage            *dockerfile.Stage
		cacheExistsFlags []bool
		cachePulledFlags []bool
	}{
		{
			"1 cached",
			&dockerfile.Stage{
				From: dockerfile.FromDirectiveFixture("FROM alpine", "alpine", ""),
				Directives: []dockerfile.Directive{
					dockerfile.RunCommitDirectiveFixture("ls", "ls"),
				},
			},
			[]bool{false, true},
			[]bool{false, true},
		},
		{
			"1 missing",
			&dockerfile.Stage{
				From: dockerfile.FromDirectiveFixture("FROM alpine", "alpine", ""),
				Directives: []dockerfile.Directive{
					dockerfile.RunDirectiveFixture("ls", "ls"),
				},
			},
			[]bool{false, false},
			[]bool{false, false},
		},
		{
			"2 cached",
			&dockerfile.Stage{
				From: dockerfile.FromDirectiveFixture("FROM alpine", "alpine", ""),
				Directives: []dockerfile.Directive{
					dockerfile.RunCommitDirectiveFixture("ls", "ls"),
					dockerfile.RunCommitDirectiveFixture("ls", "ls"),
				},
			},
			[]bool{false, true, true},
			[]bool{false, true, true},
		},
		{
			"2 missing",
			&dockerfile.Stage{
				From: dockerfile.FromDirectiveFixture("FROM alpine", "alpine", ""),
				Directives: []dockerfile.Directive{
					dockerfile.RunDirectiveFixture("ls", "ls"),
					dockerfile.RunDirectiveFixture("ls", "ls"),
				},
			},
			[]bool{false, false, false},
			[]bool{false, false, false},
		},
		{
			"1st cached 2nd missing",
			&dockerfile.Stage{
				From: dockerfile.FromDirectiveFixture("FROM alpine", "alpine", ""),
				Directives: []dockerfile.Directive{
					dockerfile.RunCommitDirectiveFixture("ls", "ls"),
					dockerfile.RunDirectiveFixture("ls", "ls"),
				},
			},
			[]bool{false, true, false},
			[]bool{false, true, false},
		},
		{
			"2nd cached 1st missing",
			&dockerfile.Stage{
				From: dockerfile.FromDirectiveFixture("FROM alpine", "alpine", ""),
				Directives: []dockerfile.Directive{
					dockerfile.RunDirectiveFixture("ls", "ls"),
					dockerfile.RunCommitDirectiveFixture("ls", "ls"),
				},
			},
			[]bool{false, false, true},
			[]bool{false, false, true},
		},
	}

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)

			alias := tc.stage.From.Alias
			opts := &buildPlanOptions{
				forceCommit:   false,
				allowModifyFS: false,
			}

			stage, err := newBuildStage(ctx, alias, tc.stage, image.DigestPairMap{}, opts)
			require.NoError(err)

			kvStore := keyvalue.MemStore{}
			cacheMgr := cache.New(ctx.ImageStore, kvStore, registry.NoopClientFixture())

			for i, node := range stage.nodes {
				if tc.cacheExistsFlags[i] {
					cacheMgr.PushCache(node.CacheID(), _testDigestPair)
				}
			}
			require.NoError(cacheMgr.WaitForPush())

			stage.pullCacheLayers(cacheMgr)

			for i, node := range stage.nodes {
				if tc.cachePulledFlags[i] {
					require.NotNil(node.digestPairs)
				} else {
					require.Nil(node.digestPairs)
				}
			}
		})
	}
}
