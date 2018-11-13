package step

import (
	"fmt"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
)

// UserStep implements BuildStep and execute USER directive
type UserStep struct {
	*baseStep

	user string
}

// NewUserStep returns a BuildStep from give build step.
func NewUserStep(args, user string, commit bool) BuildStep {
	return &UserStep{
		baseStep: newBaseStep(User, args, commit),
		user:     user,
	}
}

// GenerateConfig generates a new image config base on config from previous step.
func (s *UserStep) GenerateConfig(ctx *context.BuildContext, imageConfig *image.Config) (*image.Config, error) {
	config, err := image.NewImageConfigFromCopy(imageConfig)
	if err != nil {
		return nil, fmt.Errorf("copy image config: %s", err)
	}
	config.Config.User = s.user
	return config, nil
}
