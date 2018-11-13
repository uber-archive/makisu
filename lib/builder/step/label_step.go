package step

import (
	"fmt"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/utils"
)

// LabelStep implements BuildStep and execute LABEL directive
type LabelStep struct {
	*baseStep

	labels map[string]string
}

// NewLabelStep returns a BuildStep from give build step.
func NewLabelStep(args string, labels map[string]string, commit bool) BuildStep {
	return &LabelStep{
		baseStep: newBaseStep(Label, args, commit),
		labels:   labels,
	}
}

// GenerateConfig generates a new image config base on config from previous step.
func (s *LabelStep) GenerateConfig(ctx *context.BuildContext, imageConfig *image.Config) (*image.Config, error) {
	config, err := image.NewImageConfigFromCopy(imageConfig)
	if err != nil {
		return nil, fmt.Errorf("copy image config: %s", err)
	}
	config.Config.Labels = utils.MergeStringMaps(config.Config.Labels, s.labels)
	return config, nil
}
