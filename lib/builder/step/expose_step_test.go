package step

import (
	"testing"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"

	"github.com/stretchr/testify/require"
)

func TestExposeStepGenerateConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	ports := []string{"80/tcp", "81/udp"}
	step := NewExposeStep("", ports, false)

	c := image.NewDefaultImageConfig()
	result, err := step.GenerateConfig(ctx, &c)
	require.NoError(err)

	for _, port := range ports {
		_, ok := result.Config.ExposedPorts[port]
		require.True(ok)
	}
}

func TestExposeStepNilConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	step := NewExposeStep("", nil, false)

	_, err := step.GenerateConfig(ctx, nil)
	require.Error(err)
}
