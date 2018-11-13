package step

import (
	"errors"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/shell"
)

// RunStep implements BuildStep and execute RUN directive
type RunStep struct {
	*baseStep

	cmd string
}

// NewRunStep returns a BuildStep from give build step.
func NewRunStep(args, cmd string, commit bool) *RunStep {
	return &RunStep{
		baseStep: newBaseStep(Run, args, commit),
		cmd:      cmd,
	}
}

// RequireOnDisk always returns true, as run steps always require the stage's
// layers to be present on disk.
func (s *RunStep) RequireOnDisk() bool { return true }

// Execute executes the step.
// It shells out to run the specified command, which might change local file system.
func (s *RunStep) Execute(ctx *context.BuildContext, modifyFS bool) error {
	if !modifyFS {
		return errors.New("attempted to execute RUN step without modifying file system")
	}
	ctx.MustScan = true
	return shell.ExecCommand(log.Infof, log.Errorf, s.workingDir, "sh", "-c", s.cmd)
}
