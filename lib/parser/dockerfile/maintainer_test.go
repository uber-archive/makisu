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

func TestNewMaintainerDirective(t *testing.T) {
	buildState := newParsingState(make(map[string]string))
	buildState.stageVars = map[string]string{"arg1": "foo", "arg2": "bar", "space": " "}

	tests := []struct {
		desc    string
		succeed bool
		input   string
		author  string
	}{
		{"simple", true, "maintainer foo bar", "foo bar"},
		{"special char", true, "maintainer test* <test@exmaple.com> &\"", "test* <test@exmaple.com> &\""},
		{"no substitution", true, "maintainer test ${arg1}${arg2} $space", "test ${arg1}${arg2} $space"},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			directive, err := newDirective(test.input, buildState)
			if test.succeed {
				require.NoError(err)
				expose, ok := directive.(*MaintainerDirective)
				require.True(ok)
				require.Equal(test.author, expose.Author)
			} else {
				require.Error(err)
			}
		})
	}
}
