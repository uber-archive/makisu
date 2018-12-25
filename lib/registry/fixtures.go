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

// PullImageConfig implements registry.Client.PullImageConfig.
func (noopClientFixture) PullImageConfig(layerDigest image.Digest) (os.FileInfo, error) {
	return nil, nil
}

// PushImageConfig implements registry.Client.PushImageConfig.
func (noopClientFixture) PushImageConfig(layerDigest image.Digest) error {
	return nil
}
