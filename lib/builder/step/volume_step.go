package step

import (
	"fmt"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/utils"
)

// VolumeStep implements BuildStep and execute VOLUME directive
type VolumeStep struct {
	*baseStep

	volumes map[string]struct{}
}

// NewVolumeStep returns a BuildStep from give build step.
func NewVolumeStep(args string, volumes []string, commit bool) BuildStep {
	v := make(map[string]struct{}, len(volumes))
	for _, volume := range volumes {
		v[volume] = struct{}{}
	}
	return &VolumeStep{
		baseStep: newBaseStep(Volume, args, commit),
		volumes:  v,
	}
}

// GenerateConfig generates a new image config base on config from previous step.
func (s *VolumeStep) GenerateConfig(ctx *context.BuildContext, imageConfig *image.Config) (*image.Config, error) {
	config, err := image.NewImageConfigFromCopy(imageConfig)
	if err != nil {
		return nil, fmt.Errorf("copy image config: %s", err)
	}
	config.Config.Volumes = utils.MergeStructMaps(config.Config.Volumes, s.volumes)
	return config, nil
}
