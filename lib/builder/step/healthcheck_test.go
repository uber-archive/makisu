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
	"time"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"

	"github.com/stretchr/testify/require"
)

func TestHealthcheckStepUpdateCtxAndConfig(t *testing.T) {
	require := require.New(t)

	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	cmd := []string{"CMD", "ls", "/"}
	d5, _ := time.ParseDuration("5s")
	d0, _ := time.ParseDuration("0s")
	step, err := NewHealthcheckStep("", d0, d5, d0, 0, cmd, false)
	require.NoError(err)

	c := image.NewDefaultImageConfig()
	result, err := step.UpdateCtxAndConfig(ctx, &c)
	require.NoError(err)
	require.Equal(result.Config.Healthcheck, &image.HealthConfig{
		Interval:    d0,
		Timeout:     d5,
		StartPeriod: d0,
		Retries:     0,
		Test:        cmd,
	})
}
