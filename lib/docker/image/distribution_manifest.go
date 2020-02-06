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
	"mime"
)

const (
	// MediaTypeManifest specifies the mediaType for the current version.
	MediaTypeManifest = "application/vnd.docker.distribution.manifest.v2+json"

	// MediaTypeConfig specifies the mediaType for the image configuration.
	MediaTypeConfig = "application/vnd.docker.container.image.v1+json"

	// MediaTypeLayer is the mediaType used for layers referenced by the manifest.
	MediaTypeLayer = "application/vnd.docker.image.rootfs.diff.tar.gzip"
)

// DistributionManifest defines a schema2 manifest. It's used for docker pull and docker push.
type DistributionManifest struct {
	// SchemaVersion is the image manifest schema that this image uses.
	SchemaVersion int `json:"schemaVersion"`

	// MediaType is the media type of this schema.
	MediaType string `json:"mediaType,omitempty"`

	// Config references the image configuration as a blob.
	Config Descriptor `json:"config"`

	// Layers lists descriptors for all referenced layers, starting from base layer.
	Layers []Descriptor `json:"layers"`
}

// Descriptor describes targeted content.
type Descriptor struct {
	// MediaType describe the type of the content.
	// All text based formats are encoded as utf-8.
	MediaType string `json:"mediaType,omitempty"`

	// Size in bytes of content.
	Size int64 `json:"size,omitempty"`

	// Digest uniquely identifies the content.
	Digest Digest `json:"digest,omitempty"`
}

// DigestPair is a pair of uncompressed digest/compressed descriptor of the same layer.
// All layers in makisu are saved in gzipped format, and that value is used in distribution
// manifest; However the uncompressed digest is needed in image configs, so they are often passed
// around in pairs.
type DigestPair struct {
	TarDigest      Digest
	GzipDescriptor Descriptor
}

// UnmarshalDistributionManifest verifies MediaType and unmarshals manifest.
func UnmarshalDistributionManifest(ctHeader string, p []byte) (DistributionManifest, Descriptor, error) {
	// Need to look up by the actual media type, not the raw contents of the header.
	// Strip semicolons and anything following them.
	var mediatype string
	if ctHeader != "" {
		var err error
		mediatype, _, err = mime.ParseMediaType(ctHeader)
		if err != nil {
			return DistributionManifest{}, Descriptor{}, err
		}
	}

	if mediatype != MediaTypeManifest {
		return DistributionManifest{},
			Descriptor{},
			fmt.Errorf("unsupported manifest mediatype: %s", mediatype)
	}

	manifest := DistributionManifest{}
	if err := json.Unmarshal(p, &manifest); err != nil {
		return DistributionManifest{}, Descriptor{}, err
	}

	digest, err := NewDigester().FromBytes(p)
	if err != nil {
		return DistributionManifest{}, Descriptor{}, err
	}
	return manifest, Descriptor{Digest: digest, Size: int64(len(p)), MediaType: MediaTypeManifest}, nil
}

// GetLayerDigests returns the list of layer digests of the image.
func (manifest DistributionManifest) GetLayerDigests() []Digest {
	digests := []Digest{}
	for _, descriptor := range manifest.Layers {
		digests = append(digests, descriptor.Digest)
	}
	return digests
}

// GetConfigDigest returns digest of the image config
func (manifest DistributionManifest) GetConfigDigest() Digest {
	return manifest.Config.Digest
}

// NewEmptyDescriptor returns a 0 value descriptor.
func NewEmptyDescriptor() Descriptor {
	return Descriptor{Digest: Digest("")}
}
