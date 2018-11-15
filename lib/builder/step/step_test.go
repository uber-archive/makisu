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
	"github.com/uber/makisu/lib/parser/dockerfile"

	"github.com/stretchr/testify/require"
)

func TestNewDockerfileStep(t *testing.T) {
	ctx, cleanup := context.BuildContextFixture()
	defer cleanup()

	t.Run("FROM", func(t *testing.T) {
		require := require.New(t)
		step := dockerfile.FromDirectiveFixture("", "image", "alias")
		_, err := NewDockerfileStep(ctx, step, "")
		require.NoError(err)
	})

	t.Run("FROM bad image", func(t *testing.T) {
		require := require.New(t)
		from := dockerfile.FromDirectiveFixture("", "image:", "")
		_, err := NewDockerfileStep(ctx, from, "")
		require.Error(err)
	})

	t.Run("RUN", func(t *testing.T) {
		require := require.New(t)
		step := dockerfile.RunDirectiveFixture("", "ls /")
		_, err := NewDockerfileStep(ctx, step, "")
		require.NoError(err)
	})

	t.Run("CMD", func(t *testing.T) {
		require := require.New(t)
		step := dockerfile.CmdDirectiveFixture("", []string{"ls", "/"})
		_, err := NewDockerfileStep(ctx, step, "")
		require.NoError(err)
	})

	t.Run("LABEL", func(t *testing.T) {
		require := require.New(t)
		step := dockerfile.LabelDirectiveFixture("", map[string]string{"key": "val"})
		_, err := NewDockerfileStep(ctx, step, "")
		require.NoError(err)
	})

	t.Run("EXPOSE", func(t *testing.T) {
		require := require.New(t)
		step := dockerfile.ExposeDirectiveFixture("", []string{"80/tcp"})
		_, err := NewDockerfileStep(ctx, step, "")
		require.NoError(err)
	})

	t.Run("COPY", func(t *testing.T) {
		require := require.New(t)
		step := dockerfile.CopyDirectiveFixture("", "", "alias", []string{"."}, "/")
		_, err := NewDockerfileStep(ctx, step, "")
		require.NoError(err)
	})

	t.Run("COPY invalid", func(t *testing.T) {
		require := require.New(t)
		step := dockerfile.CopyDirectiveFixture("", "", "alias", []string{"dir1/", "dir2/"}, "/file")
		_, err := NewDockerfileStep(ctx, step, "")
		require.Error(err)
	})

	t.Run("ENTRYPOINT", func(t *testing.T) {
		require := require.New(t)
		step := dockerfile.EntrypointDirectiveFixture("", []string{"/bin/bash"})
		_, err := NewDockerfileStep(ctx, step, "")
		require.NoError(err)
	})

	t.Run("ENV", func(t *testing.T) {
		require := require.New(t)
		step := dockerfile.EnvDirectiveFixture("", map[string]string{"key": "val"})
		_, err := NewDockerfileStep(ctx, step, "")
		require.NoError(err)
	})

	t.Run("USER", func(t *testing.T) {
		require := require.New(t)
		step := dockerfile.UserDirectiveFixture("", "user")
		_, err := NewDockerfileStep(ctx, step, "")
		require.NoError(err)
	})

	t.Run("VOLUME", func(t *testing.T) {
		require := require.New(t)
		step := dockerfile.VolumeDirectiveFixture("", []string{"/tmp:/tmp"})
		_, err := NewDockerfileStep(ctx, step, "")
		require.NoError(err)
	})

	t.Run("WORKDIR", func(t *testing.T) {
		require := require.New(t)
		step := dockerfile.WorkdirDirectiveFixture("", "/home")
		_, err := NewDockerfileStep(ctx, step, "")
		require.NoError(err)
	})

	t.Run("ADD", func(t *testing.T) {
		require := require.New(t)
		step := dockerfile.AddDirectiveFixture("", "", []string{"."}, "/")
		_, err := NewDockerfileStep(ctx, step, "")
		require.NoError(err)
	})

	t.Run("ADD invalid", func(t *testing.T) {
		require := require.New(t)
		step := dockerfile.AddDirectiveFixture("", "", []string{"dir1/", "dir2/"}, "/file")
		_, err := NewDockerfileStep(ctx, step, "")
		require.Error(err)
	})
}
