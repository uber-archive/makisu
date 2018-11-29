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

// PullClientFixture returns a new registry client fixture that can handle
// image pull requests.
func PullClientFixture(ctx *context.BuildContext, testdataDir string) (*DockerRegistryClient, error) {
	image := image.MustParseName(fmt.Sprintf("localhost:5055/%s:%s", testutil.SampleImageRepoName, testutil.SampleImageTag))
	cli := &http.Client{
		Transport: pullTransportFixture{
			image:       image,
			testdataDir: testdataDir,
		},
	}
	c := NewWithClient(ctx.ImageStore, image.GetRegistry(), image.GetRepository(), cli)
	c.config.Security.TLS.Client.Disabled = true
	return c, nil
}

type pullTransportFixture struct {
	image       image.Name
	testdataDir string
}

func (t pullTransportFixture) manifestResponse() (*http.Response, error) {
	manifest, err := os.Open(filepath.Join(t.testdataDir, "files/test_distribution_manifest"))
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
	imageConfig, err := os.Open(filepath.Join(t.testdataDir, "files/test_image_config"))
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
	layerTar, err := os.Open(filepath.Join(t.testdataDir, "files/test_layer.tar"))
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
	repoURL := fmt.Sprintf("http://%s/v2/%s", t.image.GetRegistry(), t.image.GetRepository())
	manifestURL := fmt.Sprintf("%s/manifests/%s", repoURL, t.image.GetTag())
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
