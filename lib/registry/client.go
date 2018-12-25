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

package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/juju/ratelimit"

	"github.com/uber/makisu/lib/concurrency"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/storage"
	"github.com/uber/makisu/lib/utils"
	"github.com/uber/makisu/lib/utils/httputil"
)

const (
	baseManifestQuery = "http://%s/v2/%s/manifests/%s"
	baseLayerQuery    = "http://%s/v2/%s/blobs/%s"
	baseStartQuery    = "http://%s/v2/%s/blobs/uploads/"
)

// Client is the interface through which we can interact with a docker registry. It is used when
// pulling and pushing images to that registry.
type Client interface {
	Pull(tag string) (*image.DistributionManifest, error)
	Push(tag string) error
	PullManifest(tag string) (*image.DistributionManifest, error)
	PushManifest(tag string, manifest *image.DistributionManifest) error
	PullLayer(layerDigest image.Digest) (os.FileInfo, error)
	PushLayer(layerDigest image.Digest) error
	PullImageConfig(layerDigest image.Digest) (os.FileInfo, error)
	PushImageConfig(layerDigest image.Digest) error
}

var _ Client = (*DockerRegistryClient)(nil)

// DockerRegistryClient is a real registry client that is able to push and pull images. It uses a
// storage.ImageStore to interact with images on the local filesystem.
// It implements the Client interface.
type DockerRegistryClient struct {
	config     Config
	store      storage.ImageStore
	registry   string
	repository string

	// TODO: there must be a better way to test this.
	client *http.Client
}

// New returns a new default Client.
func New(store storage.ImageStore, registry, repository string) *DockerRegistryClient {
	return newClient(store, registry, repository, nil)
}

// NewWithClient returns a new Client with a customized http.Client.
func NewWithClient(store storage.ImageStore, registry, repository string, client *http.Client) *DockerRegistryClient {
	return newClient(store, registry, repository, client)
}

func newClient(store storage.ImageStore, registry, repository string, client *http.Client) *DockerRegistryClient {
	config := Config{}
	repoConfig, ok := ConfigurationMap[registry]
	if ok {
		for repo, c := range repoConfig {
			r := regexp.MustCompile(repo)
			if r.MatchString(repository) {
				config = c
				break
			}
		}
	}
	return &DockerRegistryClient{
		config:     config.applyDefaults(),
		registry:   registry,
		repository: repository,
		store:      store,
		client:     client,
	}
}

// Pull tries to pull an image from its docker registry.
// If the pull succeeded, it would store the image in the ImageStore of the client, and returns the
// distribution manifest.
func (c DockerRegistryClient) Pull(tag string) (*image.DistributionManifest, error) {
	name := image.NewImageName(c.registry, c.repository, tag)
	log.Infof("* Started pulling image %s", name)
	starttime := time.Now()

	manifest, err := c.PullManifest(tag)
	if err != nil {
		return nil, fmt.Errorf("pull manifest: %s", err)
	}

	multiError := utils.NewMultiErrors()
	workers := concurrency.NewWorkerPool(c.config.Concurrency)
	for _, layer := range manifest.GetLayerDigests() {
		l := layer
		workers.Do(func() {
			if _, err := c.PullLayer(l); err != nil {
				multiError.Add(fmt.Errorf("pull layer %s: %s", l, err))
				workers.Stop()
				return
			}
		})
	}
	l := manifest.GetConfigDigest()
	workers.Do(func() {
		if _, err := c.PullLayer(l); err != nil {
			multiError.Add(fmt.Errorf("pull image config %s: %s", l, err))
			workers.Stop()
			return
		}
	})
	workers.Wait()
	if err := multiError.Collect(); err != nil {
		return nil, err
	}

	if err := c.saveManifest(tag, manifest); err != nil {
		return nil, fmt.Errorf("save manifest: %s", err)
	}
	log.Infof("* Finished pulling image %s in %s", name, time.Since(starttime))
	return manifest, nil
}

// Push tries to push an image to docker registry, using the ImageStore of the client.
func (c DockerRegistryClient) Push(tag string) error {
	name := image.NewImageName(c.registry, c.repository, tag)
	log.Infof("* Started pushing image %s", name)
	starttime := time.Now()

	if found, err := c.manifestExists(tag); err != nil {
		return fmt.Errorf("check manifest exists for image %s: %s", name, err)
	} else if found {
		log.Infof("* Image %s already exists, overwriting", name)
	}
	manifest, err := c.loadManifest(tag)
	if err != nil {
		return fmt.Errorf("load manifest: %s", err)
	}

	multiError := utils.NewMultiErrors()
	workers := concurrency.NewWorkerPool(c.config.Concurrency)
	for _, layer := range manifest.GetLayerDigests() {
		l := layer
		workers.Do(func() {
			if err := c.PushLayer(l); err != nil {
				multiError.Add(fmt.Errorf("push layer %s: %s", l, err))
				workers.Stop()
				return
			}
		})
	}
	l := manifest.GetConfigDigest()
	workers.Do(func() {
		if err := c.PushImageConfig(l); err != nil {
			multiError.Add(fmt.Errorf("push image config %s: %s", l, err))
			workers.Stop()
			return
		}
	})
	workers.Wait()
	if err := multiError.Collect(); err != nil {
		return err
	}

	if err := c.PushManifest(tag, manifest); err != nil {
		return fmt.Errorf("push manifest: %s", err)
	}
	log.Infof("* Finished pushing image %s in %s", name, time.Since(starttime))
	return nil
}

// PullManifest pulls docker image manifest from the docker registry.
// It does not save the manifest to the store.
func (c DockerRegistryClient) PullManifest(tag string) (*image.DistributionManifest, error) {
	opt, err := c.config.Security.GetHTTPOption(c.registry, c.repository)
	if err != nil {
		return nil, fmt.Errorf("get security opt: %s", err)
	}

	URL := fmt.Sprintf(baseManifestQuery, c.registry, c.repository, tag)
	resp, err := httputil.Send(
		"GET",
		URL,
		httputil.SendClient(c.client),
		opt,
		httputil.SendTimeout(c.config.Timeout),
		c.config.sendRetry(),
		httputil.SendAcceptedCodes(http.StatusOK, http.StatusNotFound, http.StatusBadRequest),
		httputil.SendHeaders(map[string]string{"Accept": image.MediaTypeManifest}))
	if err != nil {
		return nil, fmt.Errorf("http send error: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest {
		return nil, fmt.Errorf("manifest not found")
	} else if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bad pull manifest request resp code: %d", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read resp body: %s", err)
	}
	// Parse the manifest according to the content type.
	ctHeader := resp.Header.Get("Content-Type")
	manifest, _, err := image.UnmarshalDistributionManifest(ctHeader, body)
	if err != nil {
		return nil, fmt.Errorf("unmarshal distribution manifest: %s", err)
	}
	return &manifest, nil
}

// PushManifest pushes the manifest to the registry.
func (c DockerRegistryClient) PushManifest(tag string, manifest *image.DistributionManifest) error {
	payload, err := json.MarshalIndent(manifest, "", "   ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %s", err)
	}
	headers := map[string]string{
		"Content-Type": manifest.MediaType,
		"Host":         c.registry,
	}
	opt, err := c.config.Security.GetHTTPOption(c.registry, c.repository)
	if err != nil {
		return fmt.Errorf("get security opt: %s", err)
	}

	URL := fmt.Sprintf(baseManifestQuery, c.registry, c.repository, tag)
	resp, err := httputil.Send(
		"PUT",
		URL,
		httputil.SendClient(c.client),
		opt,
		httputil.SendTimeout(c.config.Timeout),
		c.config.sendRetry(),
		httputil.SendAcceptedCodes(http.StatusOK, http.StatusCreated),
		httputil.SendHeaders(headers),
		httputil.SendBody(bytes.NewReader(payload)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// PullLayer pulls image layer from the registry, and verifies that the contents
// of that layer match the digest of the manifest.
// If the layer already exists in the imagestore, the download is skipped.
func (c DockerRegistryClient) PullLayer(layerDigest image.Digest) (os.FileInfo, error) {
	return c.pullLayerHelper(layerDigest, false)
}

// PullImageConfig pulls image config blob from the registry.
// Same as PullLayer, with slightly different log message.
func (c DockerRegistryClient) PullImageConfig(layerDigest image.Digest) (os.FileInfo, error) {
	return c.pullLayerHelper(layerDigest, true)
}

func (c DockerRegistryClient) pullLayerHelper(
	layerDigest image.Digest, isConfig bool) (os.FileInfo, error) {

	if info, err := c.store.Layers.GetDownloadOrCacheFileStat(layerDigest.Hex()); err == nil {
		if isConfig {
			log.Infof("* Skipped pulling existing image config %s:%s", c.repository, layerDigest)
		} else {
			log.Infof("* Skipped pulling existing layer %s:%s", c.repository, layerDigest)
		}
		return info, nil
	}
	opt, err := c.config.Security.GetHTTPOption(c.registry, c.repository)
	if err != nil {
		return nil, fmt.Errorf("get security opt: %s", err)
	}

	if isConfig {
		log.Infof("* Started pulling image config %s/%s:%s", c.registry, c.repository, layerDigest)
	} else {
		log.Infof("* Started pulling layer %s/%s:%s", c.registry, c.repository, layerDigest)
	}

	URL := fmt.Sprintf(baseLayerQuery, c.registry, c.repository, string(layerDigest))
	resp, err := httputil.Send(
		"GET",
		URL,
		httputil.SendClient(c.client),
		opt,
		httputil.SendTimeout(c.config.Timeout),
		c.config.sendRetry())
	if err != nil {
		return nil, fmt.Errorf("send pull layer request %s: %s", URL, err)
	}
	defer resp.Body.Close()

	// TODO: handle concurrent download of the same file.
	if err := c.store.Layers.CreateDownloadFile(layerDigest.Hex(), 0); err != nil {
		return nil, fmt.Errorf("create layer file: %s", err)
	}
	w, err := c.store.Layers.GetDownloadFileReadWriter(layerDigest.Hex())
	if err != nil {
		return nil, fmt.Errorf("get layer file readwriter: %s", err)
	}
	defer w.Close()

	if _, err := io.Copy(w, resp.Body); err != nil {
		return nil, fmt.Errorf("copy layer file: %s", err)
	}
	if err := c.saveLayer(layerDigest); err != nil {
		return nil, fmt.Errorf("save layer file: %s", err)
	}

	info, err := c.store.Layers.GetDownloadOrCacheFileStat(layerDigest.Hex())
	if err != nil {
		return nil, fmt.Errorf("get layer stat: %s", err)
	}
	if isConfig {
		log.Infof("* Finished pulling image config %s:%s", c.repository, layerDigest.Hex())
	} else {
		log.Infof("* Finished pulling layer %s:%s", c.repository, layerDigest.Hex())
	}
	return info, nil
}

// PushLayer pushes the image layer to the registry.
func (c DockerRegistryClient) PushLayer(layerDigest image.Digest) error {
	return c.pushLayerHelper(layerDigest, false)
}

// PushImageConfig pushes image config blob to the registry.
// Same as PushLayer, with slightly different log message.
func (c DockerRegistryClient) PushImageConfig(layerDigest image.Digest) error {
	return c.pushLayerHelper(layerDigest, true)
}

func (c DockerRegistryClient) pushLayerHelper(layerDigest image.Digest, isConfig bool) error {
	if found, err := c.layerExists(layerDigest); err != nil {
		return fmt.Errorf("check layer exists: %s/%s (%s): %s", c.registry, c.repository, layerDigest, err)
	} else if found {
		if isConfig {
			log.Infof("* Skipped pushing existing image config %s:%s", c.repository, layerDigest)
		} else {
			log.Infof("* Skipped pushing existing layer %s:%s", c.repository, layerDigest)
		}
		return nil
	}
	opt, err := c.config.Security.GetHTTPOption(c.registry, c.repository)
	if err != nil {
		return fmt.Errorf("get security opt: %s", err)
	}

	URL := fmt.Sprintf(baseStartQuery, c.registry, c.repository)
	resp, err := httputil.Send(
		"POST",
		URL,
		httputil.SendClient(c.client),
		opt,
		httputil.SendTimeout(c.config.Timeout),
		c.config.sendRetry(),
		httputil.SendAcceptedCodes(http.StatusAccepted),
		httputil.SendHeaders(map[string]string{"Host": c.registry}))
	if err != nil {
		return fmt.Errorf("send start push layer request %s: %s", URL, err)
	}
	defer resp.Body.Close()
	URL = resp.Header.Get("Location")
	if URL == "" {
		return fmt.Errorf("empty layer upload URL")
	}

	if isConfig {
		log.Infof("* Started pushing image config %s", layerDigest)
	} else {
		log.Infof("* Started pushing layer %s", layerDigest)
	}
	URL, err = c.pushLayerContent(layerDigest, URL)
	if err != nil {
		return fmt.Errorf("push layer content %s: %s", layerDigest, err)
	}

	parsed, err := url.Parse(URL)
	if err != nil {
		return fmt.Errorf("failed to parse location: %s", err)
	}
	q := parsed.Query()
	q.Add("digest", string(layerDigest))
	parsed.RawQuery = q.Encode()
	if err := c.commitLayer(parsed.String()); err != nil {
		return fmt.Errorf("commit layer push %s: %s", layerDigest, err)
	}
	if isConfig {
		log.Infof("* Finished pushing image config %s", layerDigest)
	} else {
		log.Infof("* Finished pushing layer %s", layerDigest)
	}
	return nil
}

// manifestExists checks with the registry to see if an image is present and available for download.
func (c DockerRegistryClient) manifestExists(tag string) (bool, error) {
	opt, err := c.config.Security.GetHTTPOption(c.registry, c.repository)
	if err != nil {
		return false, fmt.Errorf("get security opt: %s", err)
	}

	URL := fmt.Sprintf(baseManifestQuery, c.registry, c.repository, tag)
	resp, err := httputil.Send(
		"HEAD",
		URL,
		httputil.SendClient(c.client),
		opt,
		httputil.SendTimeout(c.config.Timeout),
		c.config.sendRetry(),
		httputil.SendAcceptedCodes(http.StatusOK, http.StatusNotFound, http.StatusBadRequest))
	if err != nil {
		return false, fmt.Errorf("check manifest exists: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest {
		return false, nil
	}
	return true, nil
}

// layerExists checks with the registry to see if a layer exists and is downloadable.
func (c DockerRegistryClient) layerExists(digest image.Digest) (bool, error) {
	opt, err := c.config.Security.GetHTTPOption(c.registry, c.repository)
	if err != nil {
		return false, fmt.Errorf("get security opt: %s", err)
	}

	URL := fmt.Sprintf(baseLayerQuery, c.registry, c.repository, digest)
	resp, err := httputil.Send(
		"HEAD",
		URL,
		httputil.SendClient(c.client),
		opt,
		httputil.SendTimeout(c.config.Timeout),
		c.config.sendRetry(),
		httputil.SendAcceptedCodes(http.StatusOK, http.StatusNotFound))
	if err != nil {
		return false, fmt.Errorf("check manifest exists: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return true, nil
}

func (c DockerRegistryClient) pushLayerContent(digest image.Digest, location string) (string, error) {
	info, err := c.store.Layers.GetStoreFileStat(digest.Hex())
	if err != nil {
		return "", fmt.Errorf("get layer file stat: %s", err)
	}
	size := info.Size()
	pushChunk := c.config.PushChunk
	if pushChunk == -1 {
		pushChunk = size
	}
	start, endInclusive := int64(0), utils.Min(pushChunk-1, size-1)

	r, err := c.store.Layers.GetStoreFileReader(digest.Hex())
	if err != nil {
		return "", fmt.Errorf("get layer file reader: %s", err)
	}
	defer r.Close()

	for start < size {
		location, err = c.pushOneLayerChunk(location, start, endInclusive, r)
		if err != nil {
			return location, fmt.Errorf("push layer chunk: %s", err)
		}
		start, endInclusive = endInclusive+1, utils.Min(start+pushChunk-1, size-1)
	}
	return location, nil
}

func (c DockerRegistryClient) pushOneLayerChunk(location string, start, endIncluded int64, r io.Reader) (string, error) {
	opt, err := c.config.Security.GetHTTPOption(c.registry, c.repository)
	if err != nil {
		return "", fmt.Errorf("get security opt: %s", err)
	}
	chunckSize := endIncluded + 1 - start
	r = io.LimitReader(r, chunckSize)
	readerOptions := ratelimit.NewBucketWithRate(c.config.PushRate, 1)
	headers := map[string]string{
		"Host":           c.registry,
		"Content-Type":   "application/octet-stream",
		"Content-Length": fmt.Sprintf("%d", chunckSize),
		"Content-Range":  fmt.Sprintf("%d-%d", start, endIncluded),
	}
	resp, err := httputil.Send(
		"PATCH",
		location,
		httputil.SendClient(c.client),
		opt,
		httputil.SendTimeout(c.config.Timeout),
		c.config.sendRetry(),
		// Docker registry returns 202 but gcr returns 204 on success.
		httputil.SendAcceptedCodes(http.StatusAccepted, http.StatusNoContent),
		httputil.SendHeaders(headers),
		httputil.SendBody(ratelimit.Reader(r, readerOptions)))
	if err != nil {
		return "", fmt.Errorf("send push chunk request: %s", err)
	}
	defer resp.Body.Close()

	newLocation := resp.Header.Get("Location")
	if newLocation == "" {
		return "", fmt.Errorf("empty layer upload URL")
	}
	return newLocation, nil
}

func (c DockerRegistryClient) commitLayer(location string) error {
	opt, err := c.config.Security.GetHTTPOption(c.registry, c.repository)
	if err != nil {
		return fmt.Errorf("get security opt: %s", err)
	}

	headers := map[string]string{
		"Host":           c.registry,
		"Content-Type":   "application/octet-stream",
		"Content-Length": fmt.Sprintf("%d", 0),
	}
	resp, err := httputil.Send(
		"PUT",
		location,
		httputil.SendClient(c.client),
		opt,
		httputil.SendTimeout(c.config.Timeout),
		c.config.sendRetry(),
		// Docker registry returns 201 but gcr returns 204 on success.
		httputil.SendAcceptedCodes(http.StatusCreated, http.StatusNoContent),
		httputil.SendHeaders(headers))
	if err != nil {
		return fmt.Errorf("commit: %s", err)
	}
	defer resp.Body.Close()
	return nil
}

// saveLayer moves the layer from the download file to the permanent storage in store.
func (c DockerRegistryClient) saveLayer(layerDigest image.Digest) error {
	// Verify that the layers downloaded were correct.
	r, err := c.store.Layers.GetDownloadFileReader(layerDigest.Hex())
	if err != nil {
		return fmt.Errorf("get layer file reader: %s", err)
	}
	defer r.Close()
	if verified, err := layerDigest.Equals(r); err != nil {
		return fmt.Errorf("verify layer: %s", err)
	} else if !verified {
		return fmt.Errorf("layer digest did not match")
	}

	if err := c.store.Layers.MoveDownloadFileToStore(layerDigest.Hex()); err != nil && !os.IsExist(err) {
		return fmt.Errorf("commit layer to store: %s", err)
	}
	return nil
}

// saveManifest saves given distribution manifest into local store.
func (c DockerRegistryClient) saveManifest(tag string, manifest *image.DistributionManifest) error {
	if _, err := c.store.Manifests.GetDownloadOrCacheFileStat(c.repository, tag); err == nil {
		return nil
	}
	if err := c.store.Manifests.CreateDownloadFile(c.repository, tag, 0); err != nil {
		return fmt.Errorf("create manifest file: %s", err)
	}
	w, err := c.store.Manifests.GetDownloadFileReadWriter(c.repository, tag)
	if err != nil {
		return fmt.Errorf("create manifest file readwriter: %s", err)
	}
	defer w.Close()
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal manifest: %s", err)
	}
	if _, err := w.Write(manifestJSON); err != nil {
		return fmt.Errorf("write manifest json: %s", err)
	}
	if err := c.store.Manifests.MoveDownloadFileToStore(c.repository, tag); err != nil {
		return fmt.Errorf("commit manifest to store: %s", err)
	}
	return nil
}

// loadManifest reads distribution manifest content from local manifest store.
func (c DockerRegistryClient) loadManifest(tag string) (*image.DistributionManifest, error) {
	r, err := c.store.Manifests.GetStoreFileReader(c.repository, tag)
	if err != nil {
		return nil, fmt.Errorf("get manifest file reader: %s", err)
	}
	manifestBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %s", err)
	}

	manifest := new(image.DistributionManifest)
	if err := json.Unmarshal(manifestBytes, manifest); err != nil {
		return nil, fmt.Errorf("unmarshal manifest: %s", err)
	}
	return manifest, nil
}
