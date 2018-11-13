package step

import (
	"testing"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"

	"github.com/stretchr/testify/require"
)

func TestCmdStepGenerateConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	cmd := []string{"ls", "/"}
	step := NewCmdStep("", cmd, false)

	c := image.NewDefaultImageConfig()
	result, err := step.GenerateConfig(ctx, &c)
	require.NoError(err)
	require.Equal(result.Config.Cmd, cmd)
}

func TestCmdStepNilConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	step := NewCmdStep("", nil, false)

	_, err := step.GenerateConfig(ctx, nil)
	require.Error(err)
}
