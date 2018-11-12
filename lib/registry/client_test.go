package registry

import (
	"io/ioutil"
	"testing"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/utils/testutil"

	"github.com/stretchr/testify/require"
)

func TestPullManifest(t *testing.T) {
	require := require.New(t)
	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	p, err := PullClientFixture(ctx)
	require.NoError(err)

	// Pull manifest.
	_, err = p.PullManifest(testutil.SampleImageTag)
	require.NoError(err)
}

func TestPullImage(t *testing.T) {
	require := require.New(t)
	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	p, err := PullClientFixture(ctx)
	require.NoError(err)

	// Pull image.
	_, err = p.Pull(testutil.SampleImageTag)
	require.NoError(err)

	_, err = p.store.Layers.GetStoreFileStat("393ccd5c4dd90344c9d725125e13f636ce0087c62f5ca89050faaacbb9e3ed5b")
	require.NoError(err)

	_, err = p.store.Manifests.GetStoreFileStat(testutil.SampleImageRepoName, testutil.SampleImageTag)
	require.NoError(err)
}

func TestPullWithExistingLayer(t *testing.T) {
	require := require.New(t)
	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	p, err := PullClientFixture(ctx)
	require.NoError(err)

	// Put layer in store first.
	layerTarData, err := ioutil.ReadFile("../../testdata/files/test_layer.tar")
	require.NoError(err)
	err = ctx.ImageStore.Layers.CreateDownloadFile("393ccd5c4dd90344c9d725125e13f636ce0087c62f5ca89050faaacbb9e3ed5b", 0)
	require.NoError(err)
	w, err := ctx.ImageStore.Layers.GetDownloadFileReadWriter("393ccd5c4dd90344c9d725125e13f636ce0087c62f5ca89050faaacbb9e3ed5b")
	require.NoError(err)
	_, err = w.Write(layerTarData)
	require.NoError(err)
	require.NoError(ctx.ImageStore.Layers.MoveDownloadFileToStore("393ccd5c4dd90344c9d725125e13f636ce0087c62f5ca89050faaacbb9e3ed5b"))

	// Pull image.
	_, err = p.Pull(testutil.SampleImageTag)
	require.NoError(err)

	_, err = p.store.Layers.GetStoreFileStat("393ccd5c4dd90344c9d725125e13f636ce0087c62f5ca89050faaacbb9e3ed5b")
	require.NoError(err)
	_, err = p.store.Manifests.GetStoreFileStat(testutil.SampleImageRepoName, testutil.SampleImageTag)
	require.NoError(err)
}

func TestManifestExists(t *testing.T) {
	require := require.New(t)
	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	p, err := PullClientFixture(ctx)
	require.NoError(err)

	exists, err := p.manifestExists(testutil.SampleImageTag)
	require.NoError(err)
	require.True(exists)
}

func TestLayerExists(t *testing.T) {
	require := require.New(t)
	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	p, err := PullClientFixture(ctx)
	require.NoError(err)

	exists, err := p.layerExists("sha256:" + testutil.SampleLayerTarDigest)
	require.NoError(err)
	require.True(exists)
}

func TestPushManifest(t *testing.T) {
	require := require.New(t)
	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	p, err := PushClientFixture(ctx)
	require.NoError(err)

	require.NoError(p.PushManifest(testutil.SampleImageTag, &image.DistributionManifest{}))
}

func TestPushImage(t *testing.T) {
	require := require.New(t)
	ctx, cleanup := context.BuildContextFixtureWithSampleImage()
	defer cleanup()

	p, err := PushClientFixture(ctx)
	require.NoError(err)
	require.NoError(p.Push(testutil.SampleImageTag))
}
