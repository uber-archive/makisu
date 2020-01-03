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

func TestNewAddDirective(t *testing.T) {
	buildState := newParsingState(make(map[string]string))
	buildState.stageVars = map[string]string{"prefix": "test_", "suffix": "_test", "comma": ","}

	tests := []struct {
		desc          string
		succeed       bool
		input         string
		srcs          []string
		dst           string
		chown         string
		preserveOwner bool
	}{
		{"shell single source", true, `add src dst`, []string{"src"}, "dst", "", false},
		{"shell multi source", true, `add src1 src2 dst`, []string{"src1", "src2"}, "dst", "", false},
		{"shell substitution", true, `add src1 ${prefix}src2 dst$suffix`, []string{"src1", "test_src2"}, "dst_test", "", false},
		{"shell substitution bad", false, "add src1 ${prefix", nil, "", "", false},
		{"shell chown", true, `add --chown=user:group src dst`, []string{"src"}, "dst", "user:group", false},
		{"shell chown bad", false, `add --chown= src dst`, nil, "", "", false},
		{"shell chown substitution", true, `add --chown=${prefix}user:group src1 ${prefix}src2 dst$suffix`, []string{"src1", "test_src2"}, "dst_test", "test_user:group", false},
		{"json bad", false, `add ["src"]`, nil, "", "", false},
		{"json single source", true, `add ["src", "dst"]`, []string{"src"}, "dst", "", false},
		{"json multi source", true, `add ["src1", "src2", "dst"]`, []string{"src1", "src2"}, "dst", "", false},
		{"json substitution", true, `add ["src1"$comma "src2${suffix}", "${prefix}dst"]`, []string{"src1", "src2_test"}, "test_dst", "", false},
		{"json chown", true, `add --chown=user:group ["src", "dst"]`, []string{"src"}, "dst", "user:group", false},
		{"json chown substitution", true, `add --chown=${prefix}user:group ["src1"$comma "src2${suffix}", "${prefix}dst"]`, []string{"src1", "src2_test"}, "test_dst", "test_user:group", false},
		{"json multiple flags", false, `add --chown=user:group  --archive ["src", "dst"]`, []string{"src"}, "dst", "user:group", true},
		{"json archive", true, `add --archive  ["src", "dst"]`, []string{"src"}, "dst", "", true},
		{"json archive bad", false, `add --archive=ss  ["src", "dst"]`, []string{"src"}, "dst", "", true},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			directive, err := newDirective(test.input, buildState)
			if test.succeed {
				require.NoError(err)
				cast, ok := directive.(*AddDirective)
				require.True(ok)
				require.Equal(test.srcs, cast.Srcs)
				require.Equal(test.dst, cast.Dst)
				require.Equal(test.chown, cast.Chown)
			} else {
				require.Error(err)
			}
		})
	}
}
