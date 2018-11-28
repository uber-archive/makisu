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
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
)

func TestStopsignalStepUpdateCtxAndConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	signal := 9
	step := NewStopsignalStep("", signal, false)

	c := image.NewDefaultImageConfig()
	result, err := step.UpdateCtxAndConfig(ctx, &c)
	require.NoError(err)

	require.Equal(strconv.Itoa(signal), result.Config.StopSignal)
}
