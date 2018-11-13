package step

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/registry"

	"github.com/stretchr/testify/require"
)

func TestNewFromStep(t *testing.T) {
	t.Run("NoAlias", func(t *testing.T) {
		require := require.New(t)

		_, err := NewFromStep("", "127.0.0.1:5002/alpine:latest", "")
		require.NoError(err)
	})

	t.Run("WithAlias", func(t *testing.T) {
		require := require.New(t)

		_, err := NewFromStep("", "127.0.0.1:5002/alpine:latest", "phase1")
		require.NoError(err)
	})
}

func TestFromStepSetCacheID(t *testing.T) {
	t.Run("SameFromImageSameAlias", func(t *testing.T) {
		require := require.New(t)
		context, cleanup := context.BuildContextFixture()
		defer cleanup()

		step1, err := NewFromStep("", "127.0.0.1:5002/alpine:latest", "phase1")
		require.NoError(err)
		err = step1.SetCacheID(context, "")
		require.NoError(err)

		step2, err := NewFromStep("", "127.0.0.1:5002/alpine:latest", "phase1")
		require.NoError(err)
		err = step2.SetCacheID(context, "")
		require.NoError(err)

		require.Equal(step1.CacheID(), step2.CacheID())
	})

	t.Run("SameFromImageDifferentAlias", func(t *testing.T) {
		require := require.New(t)
		context, cleanup := context.BuildContextFixture()
		defer cleanup()

		step1, err := NewFromStep("", "127.0.0.1:5002/alpine:latest", "phase1")
		require.NoError(err)
		err = step1.SetCacheID(context, "")
		require.NoError(err)

		step2, err := NewFromStep("", "127.0.0.1:5002/alpine:latest", "phase2")
		require.NoError(err)
		err = step2.SetCacheID(context, "")
		require.NoError(err)

		require.Equal(step1.CacheID(), step2.CacheID())
	})

	t.Run("DifferentFromImage", func(t *testing.T) {
		require := require.New(t)
		context, cleanup := context.BuildContextFixture()
		defer cleanup()

		step1, err := NewFromStep("", "127.0.0.1:5002/alpine:latest", "")
		require.NoError(err)
		err = step1.SetCacheID(context, "")
		require.NoError(err)

		step2, err := NewFromStep("", "127.0.0.1:5003/alpine:latest", "")
		require.NoError(err)
		err = step2.SetCacheID(context, "")
		require.NoError(err)

		require.NotEqual(step1.CacheID(), step2.CacheID())
	})
}

func TestFromStepScratch(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	step, err := NewFromStep("", image.Scratch, "")
	require.NoError(err)
	require.Equal(image.Scratch, step.GetImage())
	require.NoError(step.Execute(ctx, true))

	// Execute with modifyfs=false.
	require.NoError(step.Execute(ctx, false))

	// Commit.
	digestPairs, err := step.Commit(ctx)
	require.NoError(err)
	require.Nil(digestPairs)

	// Generate config.
	conf, err := step.GenerateConfig(ctx, nil)
	require.Equal(image.NewDefaultImageConfig(), *conf)
}

func TestFromStepRegularFlow(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	p, err := registry.PullClientFixture(ctx, "../../../testdata")
	require.NoError(err)

	step, err := NewFromStep("", "fakeregistry.dev/library/alpine:latest", "")
	step.setRegistryClient(p)

	// Execute with modifyfs=false.
	require.NoError(step.Execute(ctx, false))

	// Commit.
	digestPairs, err := step.Commit(ctx)
	require.NoError(err)
	require.Equal(1, len(digestPairs))
	require.Equal(
		image.Digest("sha256:393ccd5c4dd90344c9d725125e13f636ce0087c62f5ca89050faaacbb9e3ed5b"),
		digestPairs[0].TarDigest)

	// Generate config.
	conf, err := step.GenerateConfig(ctx, nil)
	require.NoError(err)
	expectedConfBytes, err := ioutil.ReadFile("../../../testdata/files/test_image_config")
	require.NoError(err)
	var expectedConf image.Config
	require.NoError(json.Unmarshal(expectedConfBytes, &expectedConf))
	require.Equal(expectedConf, *conf)
}
