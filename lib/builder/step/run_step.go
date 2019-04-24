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
	"errors"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/shell"
)

// RunStep implements BuildStep and execute RUN directive
type RunStep struct {
	*baseStep

	cmd string

	// Used by the user step and the run step to determine which user should run a command (format should be <user>[:<group>] or <UID>[:<GID>], default is "" which is 0:0)
	user string
}

// NewRunStep returns a BuildStep from given arguments.
func NewRunStep(args, cmd string, commit bool) *RunStep {
	return &RunStep{
		baseStep: newBaseStep(Run, args, commit),
		cmd:      cmd,
	}
}

// RequireOnDisk always returns true, as run steps always require the stage's
// layers to be present on disk.
func (s *RunStep) RequireOnDisk() bool { return true }

// ApplyCtxAndConfig setup the user that should be used to run the command
// See ./user_step.go to see how it's set in image.Config
func (s *RunStep) ApplyCtxAndConfig(ctx *context.BuildContext, imageConfig *image.Config) error {
	// This is from ./base_step.go
	s.SetWorkingDir(ctx, imageConfig)
	s.SetEnvFromContext(ctx)

	if imageConfig == nil {
		return nil
	}

	s.user = imageConfig.Config.User
	return nil
}

// Execute executes the step.
// It shells out to run the specified command, which might change local file system.
func (s *RunStep) Execute(ctx *context.BuildContext, modifyFS bool) error {
	if !modifyFS {
		return errors.New("attempted to execute RUN step without modifying file system")
	}
	ctx.MustScan = true
	return shell.ExecCommand(log.Infof, log.Errorf, s.workingDir, s.user, "sh", "-c", s.cmd)
}
