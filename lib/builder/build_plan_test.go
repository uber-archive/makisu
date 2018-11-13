package builder

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/uber/makisu/lib/cache"
	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/parser/dockerfile"
	"github.com/uber/makisu/lib/registry"

	"github.com/stretchr/testify/require"
)

func TestBuildPlanExecution(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	target := image.NewImageName("", "testrepo", "testtag")
	envImage, err := image.ParseName("scratch")
	require.NoError(err)

	cacheMgr := cache.New(nil, target, registry.NoopClientFixture())

	from := dockerfile.FromDirectiveFixture("", envImage.String(), "")
	directives := []dockerfile.Directive{
		dockerfile.EnvDirectiveFixture("TESTENV=test", map[string]string{"TESTENV": "test"}),
		dockerfile.RunCommitDirectiveFixture("ls .", "ls ."),
		dockerfile.EnvDirectiveFixture("TESTENV=test2", map[string]string{"TESTENV": "test2"}),
		dockerfile.RunCommitDirectiveFixture("ls ..", "ls .."),
	}
	stages := []*dockerfile.Stage{{from, directives}}

	plan, err := NewBuildPlan(ctx, target, cacheMgr, stages, true, false)
	require.NoError(err)

	manifest, err := plan.Execute()
	require.NoError(err)

	r, err := ctx.ImageStore.Layers.GetStoreFileReader(manifest.Config.Digest.Hex())
	require.NoError(err)

	b, err := ioutil.ReadAll(r)
	require.NoError(err)
	var config image.Config
	require.NoError(json.Unmarshal(b, &config))
	require.Equal(2, len(config.History))
	require.Equal(2, len(config.RootFS.DiffIDs))
}

func TestBuildPlanContextDirs(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	target := image.NewImageName("", "testrepo", "testtag")
	envImage, err := image.ParseName("scratch")
	require.NoError(err)

	cacheMgr := cache.New(nil, target, registry.NoopClientFixture())

	// Valid copies from previous stage.
	from1 := dockerfile.FromDirectiveFixture("", envImage.String(), "stage1")
	from2 := dockerfile.FromDirectiveFixture("", envImage.String(), "")
	directives2 := []dockerfile.Directive{
		dockerfile.CopyDirectiveFixture("", "", "stage1", []string{"/hello"}, "/hello"),
	}
	from3 := dockerfile.FromDirectiveFixture("", envImage.String(), "")
	directives3 := []dockerfile.Directive{
		dockerfile.CopyDirectiveFixture("", "", "stage1", []string{"/hello2"}, "/hello2"),
	}
	stages := []*dockerfile.Stage{{from1, nil}, {from2, directives2}, {from3, directives3}}

	plan, err := NewBuildPlan(ctx, target, cacheMgr, stages, false, false)
	require.NoError(err)
	require.Equal(map[string][]string{"stage1": {"/hello", "/hello2"}}, plan.contextDirs)

	// Copy from nonexistent stage.
	from := dockerfile.FromDirectiveFixture("", envImage.String(), "")
	directives := []dockerfile.Directive{
		dockerfile.CopyDirectiveFixture("", "", "bad_stage", []string{"/hello"}, "/hello"),
	}
	stages = []*dockerfile.Stage{{from, directives}}

	_, err = NewBuildPlan(ctx, target, cacheMgr, stages, false, false)
	require.Error(err)

	// Copy from subsequent stage.
	from1 = dockerfile.FromDirectiveFixture("", envImage.String(), "")
	directives1 := []dockerfile.Directive{
		dockerfile.CopyDirectiveFixture("", "", "stage2", []string{"/hello"}, "/hello"),
	}
	from2 = dockerfile.FromDirectiveFixture("", envImage.String(), "stage2")
	stages = []*dockerfile.Stage{{from1, directives1}, {from2, nil}}

	_, err = NewBuildPlan(ctx, target, cacheMgr, stages, false, false)
	require.Error(err)
}

func TestBuildPlanBadRun(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	target := image.NewImageName("", "testrepo", "testtag")
	envImage, err := image.ParseName("scratch")
	require.NoError(err)

	cacheMgr := cache.New(nil, target, registry.NoopClientFixture())

	from := dockerfile.FromDirectiveFixture("", envImage.String(), "")
	directives := []dockerfile.Directive{
		dockerfile.RunDirectiveFixture("ls .", "ls ."),
		dockerfile.RunDirectiveFixture("bad_executable", "bad_executable"),
	}
	stages := []*dockerfile.Stage{{from, directives}}

	plan, err := NewBuildPlan(ctx, target, cacheMgr, stages, true, false)
	require.NoError(err)

	_, err = plan.Execute()
	require.Error(err)
}

func TestDuplicateStageAlias(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	target := image.NewImageName("", "testrepo", "testtag")
	envImage, err := image.ParseName("scratch")
	require.NoError(err)

	cacheMgr := cache.New(nil, target, registry.NoopClientFixture())

	// Same image same alias.
	from1 := dockerfile.FromDirectiveFixture("", envImage.String(), "alias")
	from2 := dockerfile.FromDirectiveFixture("", envImage.String(), "alias")
	stages := []*dockerfile.Stage{{from1, nil}, {from2, nil}}

	_, err = NewBuildPlan(ctx, target, cacheMgr, stages, false, false)
	require.Error(err)

	// Same image different alias.
	from1 = dockerfile.FromDirectiveFixture("", envImage.String(), "alias1")
	from2 = dockerfile.FromDirectiveFixture("", envImage.String(), "alias2")
	stages = []*dockerfile.Stage{{from1, nil}, {from2, nil}}

	_, err = NewBuildPlan(ctx, target, cacheMgr, stages, false, false)
	require.NoError(err)
}
