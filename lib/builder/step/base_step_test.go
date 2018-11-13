package step

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"

	"github.com/stretchr/testify/require"
)

func TestBaseStep(t *testing.T) {
	require := require.New(t)

	tmpDir, err := ioutil.TempDir("/tmp", "makisu-test")
	require.NoError(err)
	defer os.RemoveAll(tmpDir)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	step := newBaseStep(Run, "", false)

	c := image.NewDefaultImageConfig()
	c.Config.WorkingDir = tmpDir
	err = step.ApplyConfig(ctx, &c)
	require.NoError(err)
	require.Equal(step.workingDir, tmpDir)
	require.NoError(step.Execute(ctx, false))
	_, err = step.Commit(ctx)
	require.NoError(err)
	require.Equal(Run, step.directive)
	require.NotEqual("", step.String())
	alias, dirs := step.ContextDirs()
	require.Equal("", alias)
	require.Len(dirs, 0)
}

func TestBaseStepNilConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	step := newBaseStep(Run, "", false)

	_, err := step.GenerateConfig(ctx, nil)
	require.Error(err)
}
