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

package step

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"strings"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/registry"
	"github.com/uber/makisu/lib/storage"
	"github.com/uber/makisu/lib/tario"
	"github.com/uber/makisu/lib/utils"
)

const (
	defaultAuthor       = "ubuild"
	defaultArchitecture = "amd64"
	defaultOS           = "linux"
)

// FromStep implements BuildStep and execute FROM directive
type FromStep struct {
	*baseStep

	image string
	alias string

	manifest *image.DistributionManifest
	client   registry.Client
}

// NewFromStep returns a BuildStep from given arguments.
func NewFromStep(args, imageName, alias string) (*FromStep, error) {
	if !strings.EqualFold(imageName, image.Scratch) {
		image, err := image.ParseNameForPull(imageName)
		if err != nil || !image.IsValid() {
			return nil, fmt.Errorf("Invalid image name: %s", imageName)
		}
		imageName = image.String()
	}
	return &FromStep{
		baseStep: newBaseStep(From, args, false),
		image:    imageName,
		alias:    alias,
	}, nil
}

// GetImage returns the image name in From step.
func (s *FromStep) GetImage() string {
	return s.image
}

// GetAlias returns stage alias defined in From step.
func (s *FromStep) GetAlias() string {
	return s.alias
}

// SetCacheID sets the cacheID of the step using the name of the base image.
// TODO: Use the sha of that image instead of the image name itself.
func (s *FromStep) SetCacheID(ctx *context.BuildContext, seed string) error {
	checksum := crc32.ChecksumIEEE([]byte(seed + string(s.directive) + s.image))
	s.cacheID = fmt.Sprintf("%x", checksum)
	return nil
}

// TODO: Not an ideal way to test. Move to build context.
func (s *FromStep) setRegistryClient(client registry.Client) {
	if s.client == nil {
		s.client = client
	}
}

// Execute updates the memFS with the FROM image. If modifyFS is true, also
// unpacks it to the local filesystem.
func (s *FromStep) Execute(ctx *context.BuildContext, modifyFS bool) error {
	if isScratch(s.image) {
		// Build from scratch, nothing to untar.
		log.Infof("Scratch base image detected")
		return nil
	}

	// Otherwise, pull image.
	manifest, err := s.getManifest(ctx.ImageStore)
	if err != nil {
		return fmt.Errorf("get manifest: %s", err)
	}

	config, err := s.getConfig(manifest.Config, ctx.ImageStore)
	if err != nil {
		return fmt.Errorf("get config: %s", err)
	}

	if config.RootFS.DiffIDs == nil || manifest.Layers == nil {
		return fmt.Errorf("empty layer digests or descriptors: %s", err)
	} else if len(config.RootFS.DiffIDs) != len(manifest.Layers) {
		return fmt.Errorf("layer digests and descriptors count doesn't match: %s", err)
	}

	// Apply each layer to the memFS.
	// If modifyFS is true, writes it to the local file system.
	for i := range config.RootFS.DiffIDs {
		descriptor := manifest.Layers[i]
		reader, err := ctx.ImageStore.Layers.GetStoreFileReader(descriptor.Digest.Hex())
		if err != nil {
			return fmt.Errorf("get reader from layer: %s", err)
		}
		gzipReader, err := tario.NewGzipReader(reader)
		if err != nil {
			return fmt.Errorf("create gzip reader for layer: %s", err)
		}
		log.Infof("* Processing FROM layer %s", descriptor.Digest.Hex())
		err = ctx.MemFS.UpdateFromTarReader(tar.NewReader(gzipReader), modifyFS)
		if err != nil {
			return fmt.Errorf("untar reader: %s", err)
		}
	}
	return nil
}

// Commit generates an image layer.
func (s *FromStep) Commit(ctx *context.BuildContext) ([]*image.DigestPair, error) {
	if isScratch(s.image) {
		return nil, nil
	}

	manifest, err := s.getManifest(ctx.ImageStore)
	if err != nil {
		return nil, fmt.Errorf("get manifest: %s", err)
	}

	config, err := s.getConfig(manifest.Config, ctx.ImageStore)
	if err != nil {
		return nil, fmt.Errorf("get config: %s", err)
	}

	if config.RootFS.DiffIDs == nil || manifest.Layers == nil {
		return nil, fmt.Errorf("empty layer digests or descriptors: %s", err)
	} else if len(config.RootFS.DiffIDs) != len(manifest.Layers) {
		return nil, fmt.Errorf("layer digests and descriptors count doesn't match: %s", err)
	}

	digestPairs := make([]*image.DigestPair, len(config.RootFS.DiffIDs))
	for i := range config.RootFS.DiffIDs {
		digestPairs[i] = &image.DigestPair{
			TarDigest:      config.RootFS.DiffIDs[i],
			GzipDescriptor: manifest.Layers[i],
		}
	}

	return digestPairs, nil
}

// UpdateCtxAndConfig updates mutable states in build context, and generates a
// new image config base on config from previous step.
func (s *FromStep) UpdateCtxAndConfig(
	ctx *context.BuildContext, imageConfig *image.Config) (*image.Config, error) {

	if isScratch(s.image) {
		config := image.NewDefaultImageConfig()
		return &config, nil
	}

	manifest, err := s.getManifest(ctx.ImageStore)
	if err != nil {
		return nil, fmt.Errorf("get manifest: %s", err)
	}

	config, err := s.getConfig(manifest.Config, ctx.ImageStore)
	if err != nil {
		return nil, fmt.Errorf("get config: %s", err)
	}

	// Update in-memory map of merged stage vars from ARG and ENV.
	envMap := utils.ConvertStringSliceToMap(config.Config.Env)
	for k, v := range envMap {
		ctx.StageVars[k] = v
	}

	return config, nil
}

func (s *FromStep) getManifest(store storage.ImageStore) (*image.DistributionManifest, error) {
	if s.manifest != nil {
		return s.manifest, nil
	}

	// Pull image.
	pullImage, err := image.ParseNameForPull(s.image)
	if err != nil {
		return nil, fmt.Errorf("parse pull image %s: %s", pullImage, err)
	}
	s.setRegistryClient(registry.New(store, pullImage.GetRegistry(), pullImage.GetRepository()))
	manifest, err := s.client.Pull(pullImage.GetTag())
	if err != nil {
		return nil, fmt.Errorf("pull image %s: %s", s.image, err)
	}
	s.manifest = manifest
	return manifest, nil
}

func (s *FromStep) getConfig(configDigest image.Descriptor, imageStore storage.ImageStore) (*image.Config, error) {
	r, err := imageStore.Layers.GetStoreFileReader(configDigest.Digest.Hex())
	if err != nil {
		return nil, fmt.Errorf("get config file reader %s: %s", configDigest.Digest.Hex(), err)
	}
	configBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read config file %s: %s", configDigest.Digest.Hex(), err)
	}
	config := new(image.Config)
	if err := json.Unmarshal(configBytes, config); err != nil {
		return nil, fmt.Errorf("unmarshal config file %s: %s", configDigest.Digest.Hex(), err)
	}
	return config, nil
}

func isScratch(i string) bool {
	return strings.EqualFold(i, image.Scratch) || strings.EqualFold(i, image.Scratch+":latest")
}
