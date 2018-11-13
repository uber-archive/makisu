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

// NewExposeStep returns a BuildStep from give build step.
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

// GenerateConfig generates a new image config base on config from previous step.
func (s *ExposeStep) GenerateConfig(ctx *context.BuildContext, imageConfig *image.Config) (*image.Config, error) {
	config, err := image.NewImageConfigFromCopy(imageConfig)
	if err != nil {
		return nil, fmt.Errorf("copy image config: %s", err)
	}
	config.Config.ExposedPorts = utils.MergeStructMaps(config.Config.ExposedPorts, s.exposedPorts)
	return config, nil
}
