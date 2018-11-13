package step

import (
	"testing"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"

	"github.com/stretchr/testify/require"
)

func TestLabelStepGenerateConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	labels := map[string]string{"key1": "val1", "key2": "val2"}
	step := NewLabelStep("", labels, false)

	c := image.NewDefaultImageConfig()
	result, err := step.GenerateConfig(ctx, &c)
	require.NoError(err)

	for k, v := range labels {
		configLabel, ok := result.Config.Labels[k]
		require.True(ok)
		require.Equal(v, configLabel)
	}
}

func TestLabelStepNilConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	step := NewLabelStep("", nil, false)

	_, err := step.GenerateConfig(ctx, nil)
	require.Error(err)
}
