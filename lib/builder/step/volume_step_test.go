package step

import (
	"testing"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"

	"github.com/stretchr/testify/require"
)

func TestVolumeStepGenerateConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	volumes := []string{"/tmp:/tmp", "/home:/home"}
	step := NewVolumeStep("", volumes, false)

	c := image.NewDefaultImageConfig()
	result, err := step.GenerateConfig(ctx, &c)
	require.NoError(err)

	for _, volume := range volumes {
		_, ok := result.Config.Volumes[volume]
		require.True(ok)
	}
}

func TestVolumeStepNilConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	step := NewVolumeStep("", nil, false)

	_, err := step.GenerateConfig(ctx, nil)
	require.Error(err)
}
