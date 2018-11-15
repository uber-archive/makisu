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

func TestNewExposeDirective(t *testing.T) {
	buildState := newParsingState(make(map[string]string))
	buildState.stageVars = map[string]string{"port": "80", "protocol": "udp", "space": " "}

	tests := []struct {
		desc    string
		succeed bool
		input   string
		ports   []string
	}{
		{"protocol", true, "expose 80/udp", []string{"80/udp"}},
		{"no protocol", true, "expose 80", []string{"80"}},
		{"multiple", true, "expose 80/udp 81", []string{"80/udp", "81"}},
		{"substitution", true, "expose ${port}${space}81/$protocol", []string{"80", "81/udp"}},
		{"bad substitution", false, "expose ${port${space}81/$protocol", nil},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			directive, err := newDirective(test.input, buildState)
			if test.succeed {
				require.NoError(err)
				expose, ok := directive.(*ExposeDirective)
				require.True(ok)
				require.Equal(test.ports, expose.Ports)
			} else {
				require.Error(err)
			}
		})
	}
}
