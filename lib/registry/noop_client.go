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
	"os"

	"github.com/uber/makisu/lib/docker/image"
)

// NoopClient implements the registry.Client interface. It returns the empty
// distribution manifest on a pull and does nothing on a push.
type NoopClient struct{}

// NoopClient inits a new NoopClient object for testing.
func NewNoopClient() Client {
	return &NoopClient{}
}

// PullImage implements registry.Client.PullImage.
func (NoopClient) Pull(tag string) (*image.DistributionManifest, error) {
	return nil, nil
}

// PushImage implements registry.Client.PushImage.
func (NoopClient) Push(tag string) error {
	return nil
}

// PullManifest pulls docker image manifest from the docker registry.
func (NoopClient) PullManifest(tag string) (*image.DistributionManifest, error) {
	return nil, nil
}

// PushManifest pushes the manifest to the registry.
func (NoopClient) PushManifest(tag string, manifest *image.DistributionManifest) error {
	return nil
}

// PullLayer implements registry.Client.PullLayer.
func (NoopClient) PullLayer(layerDigest image.Digest) (os.FileInfo, error) {
	return nil, nil
}

// PushLayer implements registry.Client.PushLayer.
func (NoopClient) PushLayer(layerDigest image.Digest) error {
	return nil
}

// PullImageConfig implements registry.Client.PullImageConfig.
func (NoopClient) PullImageConfig(layerDigest image.Digest) (os.FileInfo, error) {
	return nil, nil
}

// PushImageConfig implements registry.Client.PushImageConfig.
func (NoopClient) PushImageConfig(layerDigest image.Digest) error {
	return nil
}
