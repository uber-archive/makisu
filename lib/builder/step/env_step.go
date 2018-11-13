package step

import (
	"fmt"
	"os"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/utils"
)

// EnvStep implements BuildStep and execute ENV directive
type EnvStep struct {
	*baseStep

	envs map[string]string
}

// NewEnvStep returns a BuildStep from give build step.
func NewEnvStep(args string, envs map[string]string, commit bool) BuildStep {
	return &EnvStep{
		baseStep: newBaseStep(Env, args, commit),
		envs:     envs,
	}
}

// GenerateConfig generates a new image config base on config from previous step.
func (s *EnvStep) GenerateConfig(ctx *context.BuildContext, imageConfig *image.Config) (*image.Config, error) {
	config, err := image.NewImageConfigFromCopy(imageConfig)
	if err != nil {
		return nil, fmt.Errorf("copy image config: %s", err)
	}

	expandedEnvs := make(map[string]string, len(s.envs))
	for k, v := range s.envs {
		expandedEnvs[k] = os.ExpandEnv(v)
	}
	config.Config.Env = utils.MergeEnv(config.Config.Env, expandedEnvs)
	return config, nil
}
