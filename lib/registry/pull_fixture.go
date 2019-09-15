package registry

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/utils/testutil"
)

const (
	_testFileDirAlpine    = "../../testdata/files/alpine"
	_testFileDirAlpineDup = "../../testdata/files/alpine_dup"
)

// PullClientFixture returns a new registry client fixture that can handle image
// pull requests using a local alpine test image.
func PullClientFixtureWithAlpine(ctx *context.BuildContext) (*DockerRegistryClient, error) {
	return PullClientFixture(ctx,
		filepath.Join(_testFileDirAlpine, "test_distribution_manifest"),
		filepath.Join(_testFileDirAlpine, "test_image_config"),
		filepath.Join(_testFileDirAlpine, "test_layer.tar"))
}

// PullClientFixture returns a new registry client fixture that can handle image
// pull requests using a local alpine test image that contains duplicate layers.
func PullClientFixtureWithAlpineDup(ctx *context.BuildContext) (*DockerRegistryClient, error) {
	return PullClientFixture(ctx,
		filepath.Join(_testFileDirAlpineDup, "test_distribution_manifest"),
		filepath.Join(_testFileDirAlpine, "test_image_config"),
		filepath.Join(_testFileDirAlpine, "test_layer.tar"))
}

// PullClientFixture returns a new registry client fixture that can handle image
// pull requests.
func PullClientFixture(
	ctx *context.BuildContext, manifestPath, imageConfigPath, layerTarPath string,
) (*DockerRegistryClient, error) {

	imageName := image.MustParseName(
		fmt.Sprintf("localhost:5055/%s:%s", testutil.SampleImageRepoName, testutil.SampleImageTag))
	cli := &http.Client{
		Transport: pullTransportFixture{
			imageName:       imageName,
			manifestPath:    manifestPath,
			imageConfigPath: imageConfigPath,
			layerTarPath:    layerTarPath,
		},
	}
	c := NewWithClient(ctx.ImageStore, imageName.GetRegistry(), imageName.GetRepository(), cli)
	c.config.Security.TLS.Client.Disabled = true
	return c, nil
}

type pullTransportFixture struct {
	imageName       image.Name
	manifestPath    string
	imageConfigPath string
	layerTarPath    string
}

func (t pullTransportFixture) manifestResponse() (*http.Response, error) {
	manifest, err := os.Open(t.manifestPath)
	if err != nil {
		return nil, err
	}
	header := make(http.Header)
	header.Add("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       manifest,
		Header:     header,
	}, nil
}

func (t pullTransportFixture) imageConfigResponse() (*http.Response, error) {
	imageConfig, err := os.Open(t.imageConfigPath)
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       imageConfig,
		Header:     make(http.Header),
	}, nil
}

func (t pullTransportFixture) layerResponse() (*http.Response, error) {
	layerTar, err := os.Open(t.layerTarPath)
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       layerTar,
		Header:     make(http.Header),
	}, nil
}

func (t pullTransportFixture) RoundTrip(r *http.Request) (*http.Response, error) {
	repoURL := fmt.Sprintf(
		"http://%s/v2/%s", t.imageName.GetRegistry(), t.imageName.GetRepository())
	manifestURL := fmt.Sprintf(
		"%s/manifests/%s", repoURL, t.imageName.GetTag())
	imageConfigURL := repoURL + "/blobs/sha256:" + testutil.SampleImageConfigDigest
	layerTarURL := repoURL + "/blobs/sha256:" + testutil.SampleLayerTarDigest
	url := r.URL.String()

	if r.Method == "HEAD" {
		if url == manifestURL || url == imageConfigURL || url == layerTarURL {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
				Header:     make(http.Header),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Header:     make(http.Header),
		}, nil
	}

	if r.URL.String() == manifestURL {
		return t.manifestResponse()
	} else if r.URL.String() == imageConfigURL {
		return t.imageConfigResponse()
	} else if r.URL.String() == layerTarURL {
		return t.layerResponse()
	}

	return &http.Response{
		StatusCode: http.StatusNotFound,
		Header:     make(http.Header),
	}, nil
}
