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
	"testing"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"

	"github.com/stretchr/testify/require"
)

func TestArgStepUpdateCtx(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	val := "val"
	step := NewArgStep("", "key", &val, false)

	c := image.NewDefaultImageConfig()
	_, err := step.GenerateConfig(ctx, &c)
	require.NoError(err)
	require.Equal(val, ctx.StageVars["key"])
}

func TestArgStepNilActualVal(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	step := NewArgStep("", "key", nil, false)

	c := image.NewDefaultImageConfig()
	_, err := step.GenerateConfig(ctx, &c)
	require.NoError(err)
	_, ok := ctx.StageVars["key"]
	require.False(ok)
}
