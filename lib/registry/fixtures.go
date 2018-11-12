package registry

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/utils/testutil"
)

// NoopClientFixture implements the registry.Client interface. It returns the empty
// distribution manifest on a pull and does nothing on a push.
type noopClientFixture struct{}

// NoopClientFixture inits a new NoopClientFixture object for testing.
func NoopClientFixture() Client {
	return &noopClientFixture{}
}

// PullImage implements registry.Client.PullImage.
func (noopClientFixture) Pull(tag string) (*image.DistributionManifest, error) {
	return nil, nil
}

// PushImage implements registry.Client.PushImage.
func (noopClientFixture) Push(tag string) error {
	return nil
}

// PullManifest pulls docker image manifest from the docker registry.
func (noopClientFixture) PullManifest(tag string) (*image.DistributionManifest, error) {
	return nil, nil
}

// PushManifest pushes the manifest to the registry.
func (noopClientFixture) PushManifest(tag string, manifest *image.DistributionManifest) error {
	return nil
}

// PullLayer implements registry.Client.PullLayer.
func (noopClientFixture) PullLayer(layerDigest image.Digest) (os.FileInfo, error) {
	return nil, nil
}

// PushLayer implements registry.Client.PushLayer.
func (noopClientFixture) PushLayer(layerDigest image.Digest) error {
	return nil
}

// PullClientFixture returns a new registry client fixture that can handle
// image pull requests.
func PullClientFixture(ctx *context.BuildContext) (*DockerRegistryClient, error) {
	image := image.MustParseName(fmt.Sprintf("localhost:5055/%s:%s", testutil.SampleImageRepoName, testutil.SampleImageTag))
	cli := &http.Client{
		Transport: pullTransportFixture{image},
	}
	return NewWithClient(ctx.ImageStore, image.GetRegistry(), image.GetRepository(), cli), nil
}

type pullTransportFixture struct {
	image image.Name
}

func (t pullTransportFixture) RoundTrip(r *http.Request) (*http.Response, error) {
	manifest, err := os.Open("../../testdata/files/test_distribution_manifest")
	if err != nil {
		return nil, err
	}
	imageConfig, err := os.Open("../../testdata/files/test_image_config")
	if err != nil {
		return nil, err
	}
	layerTar, err := os.Open("../../testdata/files/test_layer.tar")
	if err != nil {
		return nil, err
	}

	repoURL := fmt.Sprintf("http://%s/v2/%s", t.image.GetRegistry(), t.image.GetRepository())
	manifestURL := fmt.Sprintf("%s/manifests/%s", repoURL, t.image.GetTag())
	imageConfigURL := repoURL + "/blobs/sha256:" + testutil.SampleImageConfigDigest
	layerTarURL := repoURL + "/blobs/sha256:" + testutil.SampleLayerTarDigest

	if r.Method == "HEAD" && r.URL.String() == manifestURL {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		}, nil
	} else if r.Method == "HEAD" && r.URL.String() == imageConfigURL {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		}, nil
	} else if r.Method == "HEAD" && r.URL.String() == layerTarURL {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		}, nil
	} else if r.Method == "GET" && r.URL.String() == manifestURL {
		header := make(http.Header)
		header.Add("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       manifest,
			Header:     header,
		}, nil
	} else if r.Method == "GET" && r.URL.String() == imageConfigURL {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       imageConfig,
			Header:     make(http.Header),
		}, nil
	} else if r.Method == "GET" && r.URL.String() == layerTarURL {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       layerTar,
			Header:     make(http.Header),
		}, nil
	}

	return &http.Response{
		StatusCode: http.StatusNotFound,
		Header:     make(http.Header),
	}, nil
}

// PushClientFixture returns a new registry client fixture that can handle
// image push requests.
func PushClientFixture(ctx *context.BuildContext) (*DockerRegistryClient, error) {
	image := image.MustParseName(fmt.Sprintf("localhost:5055/%s:%s", testutil.SampleImageRepoName, testutil.SampleImageTag))
	cli := &http.Client{
		Transport: pushTransportFixture{image},
	}
	return NewWithClient(ctx.ImageStore, image.GetRegistry(), image.GetRepository(), cli), nil
}

type pushTransportFixture struct {
	image image.Name
}

func (t pushTransportFixture) RoundTrip(r *http.Request) (*http.Response, error) {
	repoURL := fmt.Sprintf("http://%s/v2/%s", t.image.GetRegistry(), t.image.GetRepository())
	manifestURL := fmt.Sprintf("%s/manifests/%s", repoURL, t.image.GetTag())
	imageConfigURL := repoURL + "/blobs/sha256:" + testutil.SampleImageConfigDigest
	layerTarURL := repoURL + "/blobs/sha256:" + testutil.SampleLayerTarDigest
	startUploadURL := repoURL + "/blobs/uploads/"
	chunkUploadURL := repoURL + "/blobs/uploads/upload123"
	commitUploadURL := repoURL + "/blobs/uploads/commit123"
	imageConfigCommitUploadURL := commitUploadURL +
		"?digest=sha256%3A" + testutil.SampleImageConfigDigest
	layerTarCommitUploadURL := commitUploadURL +
		"?digest=sha256%3A" + testutil.SampleLayerTarDigest

	if r.Method == "HEAD" && r.URL.String() == manifestURL {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		}, nil
	} else if r.Method == "HEAD" && r.URL.String() == imageConfigURL {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		}, nil
	} else if r.Method == "HEAD" && r.URL.String() == layerTarURL {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		}, nil
	} else if r.Method == "PUT" && r.URL.String() == manifestURL {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		}, nil
	} else if r.Method == "POST" && r.URL.String() == startUploadURL {
		header := make(http.Header)
		header.Add("Location", chunkUploadURL)
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     header,
		}, nil
	} else if r.Method == "PATCH" && r.URL.String() == chunkUploadURL {
		header := make(http.Header)
		header.Add("Location", commitUploadURL)
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     header,
		}, nil
	} else if r.Method == "PUT" && r.URL.String() == imageConfigCommitUploadURL {
		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		}, nil
	} else if r.Method == "PUT" && r.URL.String() == layerTarCommitUploadURL {
		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		}, nil
	}

	return &http.Response{
		StatusCode: http.StatusNotFound,
		Header:     make(http.Header),
	}, nil
}
