package step

import (
	"path/filepath"
	"testing"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"

	"github.com/stretchr/testify/require"
)

func TestWorkdirStepGenerateConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	workdir := "/home"
	step := NewWorkdirStep("", workdir, false)

	c := image.NewDefaultImageConfig()
	result, err := step.GenerateConfig(ctx, &c)
	require.NoError(err)
	require.Equal(result.Config.WorkingDir, filepath.Join(ctx.RootDir, workdir))
}

func TestWorkdirStepNilConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	step := NewWorkdirStep("", "", false)

	_, err := step.GenerateConfig(ctx, nil)
	require.Error(err)
}
