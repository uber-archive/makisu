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
)

// CmdStep implements BuildStep and execute CMD directive
// There are three forms of command:
// CMD ["executable","param1","param2"] -> CmdStep.cmds = []string{`["executable","param1","param2"]`}
// CMD ["param1","param2"] -> CmdStep.cmds = []string{`["param1","param2"]`}
// CMD command param1 param2 -> CmdStep.cmds = []string{"command", "param1", "param2"}
type CmdStep struct {
	*baseStep

	cmd []string
}

// NewCmdStep returns a BuildStep given ParsedLine.
func NewCmdStep(args string, cmd []string, commit bool) BuildStep {
	return &CmdStep{
		baseStep: newBaseStep(Cmd, args, commit),
		cmd:      cmd,
	}
}

// UpdateCtxAndConfig updates mutable states in build context, and generates a
// new image config base on config from previous step.
func (s *CmdStep) UpdateCtxAndConfig(
	ctx *context.BuildContext, imageConfig *image.Config) (*image.Config, error) {

	config, err := image.NewImageConfigFromCopy(imageConfig)
	if err != nil {
		return nil, fmt.Errorf("copy image config: %s", err)
	}
	config.Config.Cmd = s.cmd
	return config, nil
}
