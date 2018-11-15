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

func TestNewRunDirective(t *testing.T) {
	buildState := newParsingState(make(map[string]string))
	buildState.stageVars = map[string]string{"prefix": "test_", "suffix": "_test", "comma": ","}

	tests := []struct {
		desc    string
		succeed bool
		input   string
		cmd     string
	}{
		{"good json", true, `run ["this", "cmd"]`, "this cmd"},
		{"substitution", true, `run ["${prefix}this", "cmd${suffix}"]`, "test_this cmd_test"},
		{"substitution2", true, `run ["this"$comma "cmd"]`, "this cmd"},
		{"bad substitution", false, `run ["${prefixthis", "cmd${suffix}"]`, ""},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			directive, err := newDirective(test.input, buildState)
			if test.succeed {
				require.NoError(err)
				run, ok := directive.(*RunDirective)
				require.True(ok)
				require.Equal(test.cmd, run.Cmd)
			} else {
				require.Error(err)
			}
		})
	}
}
