package registry

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/utils/testutil"
)

// PushClientFixture returns a new registry client fixture that can handle
// image push requests.
func PushClientFixture(ctx *context.BuildContext) (*DockerRegistryClient, error) {
	image := image.MustParseName(fmt.Sprintf("localhost:5055/%s:%s", testutil.SampleImageRepoName, testutil.SampleImageTag))
	cli := &http.Client{
		Transport: pushTransportFixture{image},
	}
	c := NewWithClient(ctx.ImageStore, image.GetRegistry(), image.GetRepository(), cli)
	c.config.Security.TLS.Client.Disabled = true
	return c, nil
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
	url := r.URL.String()

	resps := map[string]*http.Response{
		"HEAD" + manifestURL: {
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		},
		"HEAD" + imageConfigURL: {
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		},
		"HEAD" + layerTarURL: {
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		},
		"PUT" + manifestURL: {
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		},
		"POST" + startUploadURL: {
			StatusCode: http.StatusAccepted,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		},
		"PATCH" + chunkUploadURL: {
			StatusCode: http.StatusAccepted,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		},
		"PUT" + imageConfigCommitUploadURL: {
			StatusCode: http.StatusCreated,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		},
		"PUT" + layerTarCommitUploadURL: {
			StatusCode: http.StatusCreated,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		},
	}
	locations := map[string]string{
		"POST" + startUploadURL:  chunkUploadURL,
		"PATCH" + chunkUploadURL: commitUploadURL,
	}

	resp, found := resps[r.Method+url]
	if !found {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Header:     make(http.Header),
		}, nil
	}
	if location, found := locations[r.Method+url]; found {
		resp.Header.Add("Location", location)
	}
	return resp, nil
}
