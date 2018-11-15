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

func TestNewUserDirective(t *testing.T) {
	buildState := newParsingState(make(map[string]string))
	buildState.stageVars = map[string]string{"prefix": "test_", "suffix": "_test", "colon": ":"}

	tests := []struct {
		desc    string
		succeed bool
		input   string
		user    string
	}{
		{"too many args", false, "user u1:g1 u2:g2", ""},
		{"both", true, "user u1:g1", "u1:g1"},
		{"substitution", true, "user ${prefix}u1${colon}g1$suffix", "test_u1:g1_test"},
		{"bad substitution", false, "user ${prefix}u1${colong1$suffix", ""},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			directive, err := newDirective(test.input, buildState)
			if test.succeed {
				require.NoError(err)
				user, ok := directive.(*UserDirective)
				require.True(ok)
				require.Equal(test.user, user.User)
			} else {
				require.Error(err)
			}
		})
	}
}
