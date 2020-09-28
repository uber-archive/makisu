package registry

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/utils/testutil"
)

// PushClientFixture returns a new registry client fixture that can handle
// image push requests.
func PushClientFixture(ctx *context.BuildContext, overrides ...responseOverride) (*DockerRegistryClient, error) {
	image := image.MustParseName(fmt.Sprintf("localhost:5055/%s:%s", testutil.SampleImageRepoName, testutil.SampleImageTag))
	cli := &http.Client{
		Transport: newPushTransportFixture(image, overrides...),
	}
	c := NewWithClient(ctx.ImageStore, image.GetRegistry(), image.GetRepository(), cli)
	c.config.Security.TLS.Client.Disabled = true
	return c, nil
}

type requestTarget interface {
	getURL() string
}

type simpleRequest struct {
	url string
}

func (r simpleRequest) getURL() string {
	return r.url
}

type manifestRequest struct {
	image image.Name
}

func (r manifestRequest) getURL() string {
	return repoURL(r.image) + "/manifests/" + r.image.GetTag()
}

type layerRequest struct {
	image  image.Name
	digest image.Digest
}

func (r layerRequest) getURL() string {
	return repoURL(r.image) + "/blobs/sha256:" + r.digest.Hex()
}

type uploadRequest struct {
	image image.Name
}

func (r uploadRequest) getURL() string {
	return repoURL(r.image) + "/blobs/uploads/"
}

func (r uploadRequest) getResumeLoc() string {
	return repoURL(r.image) + "/blobs/uploads/upload123"
}

func (r uploadRequest) getCommitLoc() string {
	return repoURL(r.image) + "/blobs/uploads/commit123"
}

func (r uploadRequest) getCommitURL(digest image.Digest) string {
	return repoURL(r.image) + "/blobs/uploads/commit123?digest=sha256%3A" + digest.Hex()
}

func repoURL(image image.Name) string {
	return fmt.Sprintf("http://%s/v2/%s", image.GetRegistry(), image.GetRepository())
}

type responseOverride struct {
	Method   string
	Target   requestTarget
	Response *http.Response
}

type pushTransportFixture struct {
	image     image.Name
	responses map[string]*http.Response
	locations map[string]string
}

func newPushTransportFixture(i image.Name, overrides ...responseOverride) *pushTransportFixture {
	imageConfigDigest := image.Digest("sha256:" + testutil.SampleImageConfigDigest)
	layerTarDigest := image.Digest("sha256:" + testutil.SampleLayerTarDigest)
	manifestURL := manifestRequest{i}.getURL()
	imageConfigURL := layerRequest{i, imageConfigDigest}.getURL()
	layerTarURL := layerRequest{i, layerTarDigest}.getURL()
	upload := uploadRequest{i}
	resps := map[string]*http.Response{
		"HEAD" + manifestURL: {
			StatusCode: http.StatusNotFound,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		},
		"HEAD" + imageConfigURL: {
			StatusCode: http.StatusNotFound,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		},
		"HEAD" + layerTarURL: {
			StatusCode: http.StatusNotFound,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		},
		"PUT" + manifestURL: {
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		},
		"POST" + upload.getURL(): {
			StatusCode: http.StatusAccepted,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		},
		"PATCH" + upload.getResumeLoc(): {
			StatusCode: http.StatusAccepted,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		},
		"PUT" + upload.getCommitURL(testutil.SampleImageConfigDigest): {
			StatusCode: http.StatusCreated,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		},
		"PUT" + upload.getCommitURL(testutil.SampleLayerTarDigest): {
			StatusCode: http.StatusCreated,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		},
	}
	locs := map[string]string{
		"POST" + upload.getURL():        upload.getResumeLoc(),
		"PATCH" + upload.getResumeLoc(): upload.getCommitLoc(),
	}
	for _, o := range overrides {
		resps[strings.ToUpper(o.Method)+o.Target.getURL()] = o.Response
	}
	return &pushTransportFixture{
		image:     i,
		responses: resps,
		locations: locs,
	}
}

func (t *pushTransportFixture) RoundTrip(r *http.Request) (*http.Response, error) {
	url := r.URL.String()
	resp, found := t.responses[r.Method+url]
	if !found {
		resp = &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			Header:     make(http.Header),
		}
	}
	if location, found := t.locations[r.Method+url]; found {
		resp.Header.Add("Location", location)
	}
	resp.Request = r
	resp.Request.URL = r.URL
	return resp, nil
}
