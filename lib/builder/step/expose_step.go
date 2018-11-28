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
	"fmt"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/utils"
)

// ExposeStep implements BuildStep and execute EXPOSE directive
type ExposeStep struct {
	*baseStep

	exposedPorts map[string]struct{}
}

// NewExposeStep returns a BuildStep from given arguments.
func NewExposeStep(args string, ports []string, commit bool) BuildStep {
	exposedPorts := make(map[string]struct{}, len(ports))
	for _, port := range ports {
		exposedPorts[port] = struct{}{}
	}
	return &ExposeStep{
		baseStep:     newBaseStep(Expose, args, commit),
		exposedPorts: exposedPorts,
	}
}

// UpdateCtxAndConfig updates mutable states in build context, and generates a
// new image config base on config from previous step.
func (s *ExposeStep) UpdateCtxAndConfig(
	ctx *context.BuildContext, imageConfig *image.Config) (*image.Config, error) {

	config, err := image.NewImageConfigFromCopy(imageConfig)
	if err != nil {
		return nil, fmt.Errorf("copy image config: %s", err)
	}
	config.Config.ExposedPorts = utils.MergeStructMaps(config.Config.ExposedPorts, s.exposedPorts)
	return config, nil
}
