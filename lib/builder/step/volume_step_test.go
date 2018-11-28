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

func TestVolumeStepUpdateCtxAndConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	volumes := []string{"/tmp:/tmp", "/home:/home"}
	step := NewVolumeStep("", volumes, false)

	c := image.NewDefaultImageConfig()
	result, err := step.UpdateCtxAndConfig(ctx, &c)
	require.NoError(err)

	for _, volume := range volumes {
		_, ok := result.Config.Volumes[volume]
		require.True(ok)
	}
}

func TestVolumeStepNilConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	step := NewVolumeStep("", nil, false)

	_, err := step.UpdateCtxAndConfig(ctx, nil)
	require.Error(err)
}
