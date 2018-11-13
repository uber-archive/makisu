package step

import (
	"strings"
	"testing"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"

	"github.com/stretchr/testify/require"
)

func TestEnvStepGenerateConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	envs := map[string]string{"key": "val", "key2": "val2"}
	step := NewEnvStep("", envs, false)

	c := image.NewDefaultImageConfig()
	result, err := step.GenerateConfig(ctx, &c)
	require.NoError(err)

	for k, v := range envs {
		var found bool
		for _, env := range result.Config.Env {
			split := strings.Split(env, "=")
			require.Len(split, 2)
			if split[0] == k {
				found = true
				require.Equal(split[1], v)
			}
		}
		require.True(found)
	}
}

func TestEnvStepNilConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	step := NewEnvStep("", nil, false)

	_, err := step.GenerateConfig(ctx, nil)
	require.Error(err)
}
