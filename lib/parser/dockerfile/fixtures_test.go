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

package dockerfile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFromDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(FromDirectiveFixture("image as alias", "image", "alias"))
}

func TestRunDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(RunDirectiveFixture("ls /", "ls /"))
}

func TestCmdDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(CmdDirectiveFixture("ls /", []string{"ls", "/"}))
}

func TestLabelDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(LabelDirectiveFixture("label key=val", map[string]string{"key": "val"}))
}

func TestExposeDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(ExposeDirectiveFixture("expose 80/tcp", []string{"80/tcp"}))
}

func TestCopyDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(CopyDirectiveFixture("src1 src2 dst/", "", "", []string{"src1", "src2"}, "dst/"))
}

func TestEntrypointDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(EntrypointDirectiveFixture("ls /", []string{"ls", "/"}))
}

func TestEnvDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(EnvDirectiveFixture("key=val", map[string]string{"key": "val"}))
}

func TestUserDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(UserDirectiveFixture("user", "user"))
}

func TestVolumeDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(VolumeDirectiveFixture("volume /tmp:/tmp", []string{"/tmp:/tmp"}))
}

func TestWorkdirDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(WorkdirDirectiveFixture("/home", "/home"))
}

func TestAddDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(AddDirectiveFixture("src1 src2 dst/", "", []string{"src1", "src2"}, "dst/"))
}
