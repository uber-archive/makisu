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
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/uber/makisu/lib/utils"
)

// Docker hub defaults.
const (
	DockerHubRegistry  = "index.docker.io"
	DockerHubNamespace = "library"
	Scratch            = "scratch"
)

// Name is the identifier of an image
type Name struct {
	registry   string
	repository string
	tag        string
}

// NewImageName returns a new image name given a registry, repo and tag.
func NewImageName(registry, repo, tag string) Name {
	rawName := fmt.Sprintf("%s:%s", repo, tag)
	if registry != "" {
		rawName = filepath.Join(registry, rawName)
	}
	return MustParseName(rawName)
}

// WithRegistry makes a copy of the image name and sets the registry.
func (name Name) WithRegistry(registry string) Name {
	name.registry = registry
	return name
}

// GetRepository returns image repository
func (name Name) GetRepository() string {
	if name.repository == Scratch {
		return Scratch
	}
	repo := name.repository
	return repo
}

// GetTag returns image tag
func (name Name) GetTag() string {
	return name.tag
}

// GetRegistry returns image repository
func (name Name) GetRegistry() string {
	return name.registry
}

// IsValid returns whether or not the image name is valid, that is,
// an image name needs to have a non-empty registry, repository and tag
func (name Name) IsValid() bool {
	if name.repository == Scratch {
		return true
	}
	return name.registry != "" && name.repository != "" && name.tag != ""
}

// ShortName returns the name of the image without the registry information
func (name Name) ShortName() string {
	return fmt.Sprintf("%s:%s", name.GetRepository(), name.tag)
}

// String returns the full name of the image with the registry information if available
func (name Name) String() string {
	if name.repository == Scratch {
		return name.ShortName()
	}
	return filepath.Join(name.registry, name.ShortName())
}

// ParseName parses image name of format <registry>/<repo>:<tag>
func ParseName(input string) (Name, error) {
	result := Name{
		registry:   "",
		repository: input,
		tag:        "latest",
	}

	slashIndex := strings.LastIndex(input, "/")
	sepIndex := strings.LastIndex(input, ":")
	if sepIndex < slashIndex || sepIndex == -1 {
		// <ip>:<port>/<repo>
		// <dns>:<port>/<repo>
		// <dns>/<repo>
		// <repo>
		result.repository = input
		result.tag = "latest"
	} else {
		// if sepIndex >= slashIndex && sepIndex == -1
		// <ip>:<port>/<repo>:<tag>
		// <dns>:<port>/<repo>:<tag>
		// <repo>:<tag>
		result.repository = input[:sepIndex]
		result.tag = input[sepIndex+1:]
	}

	// Separate hostname from repo.
	// It ignores the fact that - cannot be the first or last character, or that _ is not valid
	// character in URL, but those checks would make the regex too complex.
	hostnameRegexp := regexp.MustCompile(`^([\w\d\.-]+(\.[\w\d\.-]+|:[\d]+))\/`)
	parts := hostnameRegexp.FindStringSubmatch(result.repository)
	if parts != nil {
		result.registry = parts[1]
		result.repository = result.repository[len(parts[1])+1:]
	}

	return result, nil
}

// ParseNameForPull parses image name of format <registry>/<repo>:<tag>. If
// input doesn't contain registry information, apply defaults for dockerhub.
func ParseNameForPull(input string) (Name, error) {
	result, err := ParseName(input)
	if err != nil {
		return result, err
	}

	if result.repository == Scratch {
		return result, nil
	}

	// For docker hub registry.
	if result.registry == "" {
		result.registry = DockerHubRegistry
		if !strings.Contains(result.repository, "/") {
			result.repository = DockerHubNamespace + "/" + result.repository
		}
	}

	return result, nil
}

// MustParseName calls ParseName on the input and panics if the parsing
// of the image name fails
func MustParseName(input string) Name {
	name, err := ParseName(input)
	utils.Must(err == nil, "Failed to parse image name: %s", err)
	return name
}
