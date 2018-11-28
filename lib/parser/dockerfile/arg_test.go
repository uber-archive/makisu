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

func TestNewArgDirective(t *testing.T) {
	buildState := newParsingState(make(map[string]string))
	buildState.stageVars = map[string]string{"prefix": "test_", "suffix": "_test", "space": " "}

	tests := []struct {
		desc       string
		succeed    bool
		input      string
		name       string
		defaultVal string
	}{
		{"no default", true, "arg k1", "k1", ""},
		{"default", true, "arg k1=v1", "k1", "v1"},
		{"multiple kv pairs", false, "arg k1=v1 k2=v2", "", ""},
		{"multiple single args", false, "arg k1 k2", "", ""},
		{"bad args", false, `arg "k1"k2`, "", ""},
		{"quotes", true, `arg k1="v1a v1b"`, "k1", "v1a v1b"},
		{"quotes substitution", true, `arg ${prefix}k1${suffix}="${prefix}v1a${space}v1b${suffix}"`, "test_k1_test", "test_v1a v1b_test"},
		{"bad substitution", false, "arg ${prefix=val", "", ""},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			directive, err := newDirective(test.input, buildState)
			if test.succeed {
				require.NoError(err)
				arg, ok := directive.(*ArgDirective)
				require.True(ok)
				require.Equal(test.name, arg.Name)
				require.Equal(test.defaultVal, arg.DefaultVal)
			} else {
				require.Error(err)
			}
		})
	}
}
