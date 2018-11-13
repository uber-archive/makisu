package step

import (
	"fmt"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
)

// EntrypointStep implements BuildStep and execute ENTRYPOINT directive
// There are three forms of command:
// ENTRYPOINT ["executable","param1","param2"] -> EntrypointStep.entrypoint = []string{`["executable","param1","param2"]`}
// ENTRYPOINT ["param1","param2"] -> EntrypointStep.entrypoint = []string{`["param1","param2"]`}
// ENTRYPOINT command param1 param2 -> EntrypointStep.entrypoint = []string{"command", "param1", "param2"}
type EntrypointStep struct {
	*baseStep
	entrypoint []string
}

// NewEntrypointStep returns a BuildStep from give build step.
func NewEntrypointStep(args string, entrypoint []string, commit bool) BuildStep {
	return &EntrypointStep{
		baseStep:   newBaseStep(Entrypoint, args, commit),
		entrypoint: entrypoint,
	}
}

// GenerateConfig generates a new image config base on config from previous step.
func (s *EntrypointStep) GenerateConfig(ctx *context.BuildContext, imageConfig *image.Config) (*image.Config, error) {
	config, err := image.NewImageConfigFromCopy(imageConfig)
	if err != nil {
		return nil, fmt.Errorf("copy image config: %s", err)
	}
	config.Config.Entrypoint = s.entrypoint
	return config, nil
}
