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
	"path/filepath"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
)

// WorkdirStep implements BuildStep and execute WORKDIR directive
type WorkdirStep struct {
	*baseStep

	workingDir string
}

// NewWorkdirStep returns a BuildStep from given arguments.
func NewWorkdirStep(args string, workingDir string, commit bool) BuildStep {
	return &WorkdirStep{
		baseStep:   newBaseStep(Workdir, args, commit),
		workingDir: workingDir,
	}
}

// UpdateCtxAndConfig updates mutable states in build context, and generates a
// new image config base on config from previous step.
func (s *WorkdirStep) UpdateCtxAndConfig(
	ctx *context.BuildContext, imageConfig *image.Config) (*image.Config, error) {

	config, err := image.NewImageConfigFromCopy(imageConfig)
	if err != nil {
		return nil, fmt.Errorf("copy image config: %s", err)
	}

	workdir := os.ExpandEnv(s.workingDir)
	if filepath.IsAbs(workdir) {
		config.Config.WorkingDir = ctx.RootDir
	}
	config.Config.WorkingDir = filepath.Join(config.Config.WorkingDir, workdir)

	// Create this workdir if it does not exist already.
	if _, err := os.Lstat(config.Config.WorkingDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(config.Config.WorkingDir, 0755); err != nil {
				return nil, fmt.Errorf("mkdir all working dir %s: %s", config.Config.WorkingDir, err)
			}
		} else {
			return nil, fmt.Errorf("lstat working dir %s: %s", config.Config.WorkingDir, err)
		}
	}
	return config, nil
}
