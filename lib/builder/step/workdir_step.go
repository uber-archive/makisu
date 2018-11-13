package step

import (
	"fmt"
	"path/filepath"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
)

// WorkdirStep implements BuildStep and execute WORKDIR directive
type WorkdirStep struct {
	*baseStep

	workingDir string
}

// NewWorkdirStep returns a BuildStep from give build step.
func NewWorkdirStep(args string, workingDir string, commit bool) BuildStep {
	return &WorkdirStep{
		baseStep:   newBaseStep(Workdir, args, commit),
		workingDir: workingDir,
	}
}

// GenerateConfig generates a new image config base on config from previous step.
func (s *WorkdirStep) GenerateConfig(ctx *context.BuildContext, imageConfig *image.Config) (*image.Config, error) {
	config, err := image.NewImageConfigFromCopy(imageConfig)
	if err != nil {
		return nil, fmt.Errorf("copy image config: %s", err)
	}
	if filepath.IsAbs(s.workingDir) {
		config.Config.WorkingDir = ctx.RootDir
	}
	config.Config.WorkingDir = filepath.Join(config.Config.WorkingDir, s.workingDir)

	return config, nil
}
