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

func TestNewFromDirective(t *testing.T) {
	buildState := newParsingState(make(map[string]string))
	buildState.globalArgs["prefix"] = "test_"
	buildState.globalArgs["suffix"] = "_test"
	tests := []struct {
		desc    string
		succeed bool
		input   string
		image   string
		alias   string
	}{
		{"no tag, no alias", true, "from test_image", "test_image", ""},
		{"tag, no alias", true, "from test_image:trusty", "test_image:trusty", ""},
		{"no tag, alias", true, "from test_image as test_alias", "test_image", "test_alias"},
		{"tag, alias", true, "from test_image:trusty as test_alias", "test_image:trusty", "test_alias"},
		{"registry", true, "from 127.0.0.1:5050/test_image:trusty as test_alias", "127.0.0.1:5050/test_image:trusty", "test_alias"},
		{"mixed case", true, "fRoM test_image:trusty aS test_alias", "test_image:trusty", "test_alias"},
		{"too few args", false, "from", "", ""},
		{"too many args", false, "from test_image as test_alias another_arg", "", ""},
		{"missing alias", false, "from test_image:trusty as", "", ""},
		{"bad 'as'", false, "from test_image:trusty sa test_alias", "", ""},
		{"substitution", true, "from ${prefix}image:trusty as alias$suffix", "test_image:trusty", "alias_test"},
		{"bad substitution", false, "from ${prefiximage:trusty as alias$suffix", "", ""},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			directive, err := newDirective(test.input, buildState)
			if test.succeed {
				require.NoError(err)
				from, ok := directive.(*FromDirective)
				require.True(ok)
				require.Equal(test.image, from.Image)
				require.Equal(test.alias, from.Alias)
			} else {
				require.Error(err)
			}
		})
	}
}
