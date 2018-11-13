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

// GenerateConfig generates a new image config base on config from previous step.
func (s *CmdStep) GenerateConfig(ctx *context.BuildContext, imageConfig *image.Config) (*image.Config, error) {
	config, err := image.NewImageConfigFromCopy(imageConfig)
	if err != nil {
		return nil, fmt.Errorf("copy image config: %s", err)
	}
	config.Config.Cmd = s.cmd
	return config, nil
}
