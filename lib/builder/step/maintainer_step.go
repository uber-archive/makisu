package step

import (
	"fmt"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
)

// MaintainerStep implements BuildStep and execute MAINTAINER directive
type MaintainerStep struct {
	*baseStep

	author string
}

// NewMaintainerStep returns a BuildStep from give build step.
func NewMaintainerStep(args string, author string, commit bool) BuildStep {
	return &MaintainerStep{
		baseStep: newBaseStep(Maintainer, args, commit),
		author:   author,
	}
}

// GenerateConfig generates a new image config base on config from previous step.
func (s *MaintainerStep) GenerateConfig(ctx *context.BuildContext, imageConfig *image.Config) (*image.Config, error) {
	config, err := image.NewImageConfigFromCopy(imageConfig)
	if err != nil {
		return nil, fmt.Errorf("copy image config: %s", err)
	}
	config.Author = s.author
	return config, nil
}
