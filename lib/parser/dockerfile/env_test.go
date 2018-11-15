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

func TestNewEnvDirective(t *testing.T) {
	buildState := newParsingState(make(map[string]string))
	buildState.stageVars = map[string]string{"prefix": "test_", "suffix": "_test", "space": " "}

	tests := []struct {
		desc    string
		succeed bool
		input   string
		envs    map[string]string
	}{
		{"single", true, "env k1 v1", map[string]string{"k1": "v1"}},
		{"missing space", false, "env k1\tv1", nil},
		{"single spaces", true, "env k1 v1${space}v2 v3v4", map[string]string{"k1": "v1 v2 v3v4"}},
		{"single key-value", true, "env k1=v1", map[string]string{"k1": "v1"}},
		{"quotes", true, `env k1="v1a v1b"`, map[string]string{"k1": "v1a v1b"}},
		{"mutiple", true, "env k1=v1 k2=v2", map[string]string{"k1": "v1", "k2": "v2"}},
		{"substitution", true, "env k1=${prefix}v1 k2=v2$suffix", map[string]string{"k1": "test_v1", "k2": "v2_test"}},
		{"bad substitution", false, "env k1=${prefixv1 k2=v2$suffix", nil},
		{"quotes_substitution", true, `env k1="v1a${space}v1b"`, map[string]string{"k1": "v1a v1b"}},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			directive, err := newDirective(test.input, buildState)
			if test.succeed {
				require.NoError(err)
				env, ok := directive.(*EnvDirective)
				require.True(ok)
				require.Equal(test.envs, env.Envs)
			} else {
				require.Error(err)
			}
		})
	}
}
