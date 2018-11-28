//  Copyright (c) 2018 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	err = step.ApplyCtxAndConfig(ctx, &c)
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

	_, err := step.UpdateCtxAndConfig(ctx, nil)
	require.Error(err)
}
