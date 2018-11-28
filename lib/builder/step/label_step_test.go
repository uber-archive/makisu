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

func TestLabelStepUpdateCtxAndConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	labels := map[string]string{"key1": "val1", "key2": "val2"}
	step := NewLabelStep("", labels, false)

	c := image.NewDefaultImageConfig()
	result, err := step.UpdateCtxAndConfig(ctx, &c)
	require.NoError(err)

	for k, v := range labels {
		configLabel, ok := result.Config.Labels[k]
		require.True(ok)
		require.Equal(v, configLabel)
	}
}

func TestLabelStepNilConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	step := NewLabelStep("", nil, false)

	_, err := step.UpdateCtxAndConfig(ctx, nil)
	require.Error(err)
}
