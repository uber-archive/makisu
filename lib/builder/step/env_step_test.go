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
	"strings"
	"testing"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"

	"github.com/stretchr/testify/require"
)

func TestEnvStepUpdateCtxAndConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	envs := map[string]string{"key": "val", "key2": "val2"}
	step := NewEnvStep("", envs, false)

	c := image.NewDefaultImageConfig()
	result, err := step.UpdateCtxAndConfig(ctx, &c)
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

	_, err := step.UpdateCtxAndConfig(ctx, nil)
	require.Error(err)
}
