package step

import (
	"testing"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"

	"github.com/stretchr/testify/require"
)

func TestMaintainerStepGenerateConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	author := "$foo <test@example.com>"
	step := NewMaintainerStep("", author, false)

	c := image.NewDefaultImageConfig()
	result, err := step.GenerateConfig(ctx, &c)
	require.NoError(err)

	require.Equal(author, result.Author)
}

func TestMaintainerStepEmptyConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	step := NewMaintainerStep("", "", false)

	_, err := step.GenerateConfig(ctx, nil)
	require.Error(err)
}
