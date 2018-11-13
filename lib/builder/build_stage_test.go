package builder

import (
	"testing"

	"github.com/uber/makisu/lib/cache"
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

			stage, err := newBuildStage(
				ctx, tc.stage, map[string][]*image.DigestPair{}, false, false)
			require.NoError(err)

			imageName, err := image.ParseName("registry.net/repo:tag")
			require.NoError(err)
			kvstore := cache.MemKVStore{}
			cacheMgr := cache.New(kvstore, imageName, registry.NoopClientFixture())

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
