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

func TestNewCopyDirective(t *testing.T) {
	buildState := newParsingState(make(map[string]string))
	buildState.stageVars = map[string]string{"prefix": "test_", "suffix": "_test", "comma": ","}

	tests := []struct {
		desc           string
		succeed        bool
		input          string
		srcs           []string
		dst            string
		fromStage      string
		chown          string
		preserverOwner bool
	}{
		{"missing args", false, "copy ", nil, "", "", "", false},
		{"missing args after replace", false, "copy \\", nil, "", "", "", false},
		{"shell single source", true, `copy src dst`, []string{"src"}, "dst", "", "", false},
		{"shell multi source", true, `copy src1 src2 dst`, []string{"src1", "src2"}, "dst", "", "", false},
		{"shell substitution", true, `copy src1 ${prefix}src2 dst$suffix`, []string{"src1", "test_src2"}, "dst_test", "", "", false},
		{"shell substitution bad", false, "copy src1 ${prefix", nil, "", "", "", false},
		{"shell from", true, `copy --from=stage src dst`, []string{"src"}, "dst", "stage", "", false},
		{"shell from bad", false, `copy --from= src dst`, nil, "", "", "", false},
		{"shell chown", true, `copy --chown=user:group src dst`, []string{"src"}, "dst", "", "user:group", false},
		{"shell from chown", true, `copy --chown=user:group --from=stage src dst`, []string{"src"}, "dst", "stage", "user:group", false},
		{"shell from chown bad", false, `copy --chown=user:group --from= src dst`, nil, "", "", "", false},
		{"shell from chown substitution", true, `copy --chown=${prefix}user:group --from=${prefix}stage src1 ${prefix}src2 dst$suffix`, []string{"src1", "test_src2"}, "dst_test", "test_stage", "test_user:group", false},
		{"json single source", true, `copy ["src", "dst"]`, []string{"src"}, "dst", "", "", false},
		{"json multi source", true, `copy ["src1", "src2", "dst"]`, []string{"src1", "src2"}, "dst", "", "", false},
		{"json substitution", true, `copy ["src1"$comma "src2${suffix}", "${prefix}dst"]`, []string{"src1", "src2_test"}, "test_dst", "", "", false},
		{"json from", true, `copy --from=stage ["src", "dst"]`, []string{"src"}, "dst", "stage", "", false},
		{"json from missing args", false, "copy --from=stage", nil, "", "", "", false},
		{"json from substitution", true, `copy --from=${prefix}stage ["src1"$comma "src2${suffix}", "${prefix}dst"]`, []string{"src1", "src2_test"}, "test_dst", "test_stage", "", false},
		{"json chown", true, `copy --chown=user:group ["src", "dst"]`, []string{"src"}, "dst", "", "user:group", false},
		{"json from chown", true, `copy --chown=user:group --from=stage ["src", "dst"]`, []string{"src"}, "dst", "stage", "user:group", false},
		{"json from chown substitution", true, `copy --chown=${prefix}user:group --from=${prefix}stage ["src1"$comma "src2${suffix}", "${prefix}dst"]`, []string{"src1", "src2_test"}, "test_dst", "test_stage", "test_user:group", false},
		{"json multiple flags", false, `copy --chown=user:group --archive ["src", "dst"]`, []string{"src"}, "dst", "", "user:group", true},
		{"json archive", true, `copy --archive ["src", "dst"]`, []string{"src"}, "dst", "", "", true},
		{"json archive bad", false, `copy --archive==bad ["src", "dst"]`, []string{"src"}, "dst", "", "", true},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			directive, err := newDirective(test.input, buildState)
			if test.succeed {
				require.NoError(err)
				cast, ok := directive.(*CopyDirective)
				require.True(ok)
				require.Equal(test.srcs, cast.Srcs)
				require.Equal(test.dst, cast.Dst)
				require.Equal(test.fromStage, cast.FromStage)
				require.Equal(test.chown, cast.Chown)
			} else {
				require.Error(err)
			}
		})
	}
}
