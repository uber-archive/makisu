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
	"os"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/utils"
)

// EnvStep implements BuildStep and execute ENV directive
type EnvStep struct {
	*baseStep

	envs map[string]string
}

// NewEnvStep returns a BuildStep from given arguments.
func NewEnvStep(args string, envs map[string]string, commit bool) BuildStep {
	return &EnvStep{
		baseStep: newBaseStep(Env, args, commit),
		envs:     envs,
	}
}

// UpdateCtxAndConfig updates mutable states in build context, and generates a
// new image config base on config from previous step.
func (s *EnvStep) UpdateCtxAndConfig(
	ctx *context.BuildContext, imageConfig *image.Config) (*image.Config, error) {

	// Update in-memory map of merged stage vars from ARG and ENV.
	for k, v := range s.envs {
		ctx.StageVars[k] = v
	}

	// Update image config.
	config, err := image.NewImageConfigFromCopy(imageConfig)
	if err != nil {
		return nil, fmt.Errorf("copy image config: %s", err)
	}

	expandedEnvs := make(map[string]string, len(s.envs))
	for k, v := range s.envs {
		expandedEnvs[k] = os.ExpandEnv(v)
	}
	config.Config.Env = utils.MergeEnv(config.Config.Env, expandedEnvs)
	return config, nil
}
