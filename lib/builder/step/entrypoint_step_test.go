package step

import (
	"testing"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"

	"github.com/stretchr/testify/require"
)

func TestEntrypointStepGenerateConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	entrypoint := []string{"ls", "/"}
	step := NewEntrypointStep("", entrypoint, false)

	c := image.NewDefaultImageConfig()
	result, err := step.GenerateConfig(ctx, &c)
	require.NoError(err)
	require.Equal(result.Config.Entrypoint, entrypoint)
}

func TestEntrypointStepNilConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	step := NewEntrypointStep("", nil, false)

	_, err := step.GenerateConfig(ctx, nil)
	require.Error(err)
}
