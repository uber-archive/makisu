package step

import (
	"testing"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"

	"github.com/stretchr/testify/require"
)

func TestUserStepGenerateConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	user := "user"
	step := NewUserStep("", user, false)

	c := image.NewDefaultImageConfig()
	result, err := step.GenerateConfig(ctx, &c)
	require.NoError(err)
	require.Equal(result.Config.User, user)
}

func TestUserStepNilConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	step := NewUserStep("", "", false)

	_, err := step.GenerateConfig(ctx, nil)
	require.Error(err)
}
