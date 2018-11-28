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

// VolumeStep implements BuildStep and execute VOLUME directive
type VolumeStep struct {
	*baseStep

	volumes map[string]struct{}
}

// NewVolumeStep returns a BuildStep from given arguments.
func NewVolumeStep(args string, volumes []string, commit bool) BuildStep {
	v := make(map[string]struct{}, len(volumes))
	for _, volume := range volumes {
		v[volume] = struct{}{}
	}
	return &VolumeStep{
		baseStep: newBaseStep(Volume, args, commit),
		volumes:  v,
	}
}

// UpdateCtxAndConfig updates mutable states in build context, and generates a
// new image config base on config from previous step.
func (s *VolumeStep) UpdateCtxAndConfig(
	ctx *context.BuildContext, imageConfig *image.Config) (*image.Config, error) {

	config, err := image.NewImageConfigFromCopy(imageConfig)
	if err != nil {
		return nil, fmt.Errorf("copy image config: %s", err)
	}
	config.Config.Volumes = utils.MergeStructMaps(config.Config.Volumes, s.volumes)
	return config, nil
}
