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

package image

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// RootFS describes images root filesystem
type RootFS struct {
	Type    string   `json:"type"`
	DiffIDs []Digest `json:"diff_ids,omitempty"`
}

// ID is the content-addressable ID of an image.
type ID Digest

// V1Image stores the V1 image configuration.
type V1Image struct {
	// ID a unique 64 character identifier of the image
	ID string `json:"id,omitempty"`
	// Parent id of the image
	Parent string `json:"parent,omitempty"`
	// Comment user added comment
	Comment string `json:"comment,omitempty"`
	// Created timestamp when image was created
	Created time.Time `json:"created"`
	// Container is the id of the container used to commit
	Container string `json:"container,omitempty"`
	// ContainerConfiguration is the configuration of the container that is committed into the image
	// It only contains history information. It should be safe to leave this field empty.
	ContainerConfiguration *ContainerConfig `json:"container_config,omitempty"`
	// DockerVersion specifies version on which image is built
	DockerVersion string `json:"docker_version,omitempty"`
	// Author of the image
	Author string `json:"author,omitempty"`
	// Config is the configuration of the container received from the client
	Config *ContainerConfig `json:"config,omitempty"`
	// Architecture is the hardware that the image is build and runs on
	Architecture string `json:"architecture,omitempty"`
	// OS is the operating system used to build and run the image
	OS string `json:"os,omitempty"`
	// Size is the total size of the image including all layers it is composed of
	Size int64 `json:",omitempty"`
}

// Config stores the image configuration
type Config struct {
	V1Image
	Parent  ID        `json:"parent,omitempty"`
	RootFS  *RootFS   `json:"rootfs,omitempty"`
	History []History `json:"history,omitempty"`

	// rawJSON caches the immutable JSON associated with this image.
	rawJSON []byte

	// computedID is the ID computed from the hash of the image config.
	// Not to be confused with the legacy V1 ID in V1Image.
	computedID ID
}

// ID returns the image's content-addressable ID.
func (img *Config) ID() ID {
	return img.computedID
}

// MarshalJSON serializes the image to JSON.
// It sorts the top-level keys so that JSON that's been manipulated by a push/pull cycle with a
// legacy registry won't end up with a different key order.
func (img *Config) MarshalJSON() ([]byte, error) {
	type MarshalImage Config

	pass1, err := json.Marshal(MarshalImage(*img))
	if err != nil {
		return nil, err
	}

	var c map[string]*json.RawMessage
	if err := json.Unmarshal(pass1, &c); err != nil {
		return nil, err
	}
	return json.Marshal(c)
}

// History stores build commands that were used to create an image.
type History struct {
	// Created timestamp for build point
	Created time.Time `json:"created"`
	// Author of the build point
	Author string `json:"author,omitempty"`
	// CreatedBy keeps the Dockerfile command used while building image.
	CreatedBy string `json:"created_by,omitempty"`
	// Comment is custom message set by the user when creating the image.
	Comment string `json:"comment,omitempty"`
	// EmptyLayer is set to true if this history item did not generate a
	// layer. Otherwise, the history item is associated with the next
	// layer in the RootFS section.
	EmptyLayer bool `json:"empty_layer,omitempty"`
}

// Exporter provides interface for exporting and importing images.
type Exporter interface {
	Load(io.ReadCloser, io.Writer, bool) error
	// TODO: Load(net.Context, io.ReadCloser, <- chan StatusMessage) error
	Save([]string, io.Writer) error
}

// NewDefaultImageConfig returns a default image config that is used for images built from scratch.
func NewDefaultImageConfig() Config {
	return Config{
		V1Image: V1Image{
			Architecture: "amd64",
			Config: &ContainerConfig{
				Env: []string{
					"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
				},
			},
			ContainerConfiguration: &ContainerConfig{},
			DockerVersion:          "1.12.6",
			OS:                     "linux",
		},
		RootFS: &RootFS{
			Type:    "layers",
			DiffIDs: []Digest{},
		},
	}
}

// NewImageConfigFromJSON creates an Image configuration from json.
func NewImageConfigFromJSON(src []byte) (*Config, error) {
	img := &Config{}

	if err := json.Unmarshal(src, img); err != nil {
		return nil, err
	}
	if img.RootFS == nil {
		return nil, fmt.Errorf("Invalid image JSON, no RootFS key")
	}
	if img.RootFS.DiffIDs == nil {
		img.RootFS.DiffIDs = []Digest{}
	}

	img.rawJSON = src
	return img, nil
}

// NewImageConfigFromCopy returns a copy of given config.
func NewImageConfigFromCopy(imageConfig *Config) (*Config, error) {
	encoded, err := json.Marshal(imageConfig)
	if err != nil {
		return nil, fmt.Errorf("marshal image config: %s", err)
	}
	return NewImageConfigFromJSON(encoded)
}
