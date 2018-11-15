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

func TestParseKeyVals(t *testing.T) {
	tests := []struct {
		desc    string
		input   string
		succeed bool
		output  map[string]string
	}{
		{"key-value", "  a=b b=c\tc=d", true, map[string]string{"a": "b", "b": "c", "c": "d"}},
		{"key-value2", "ab=ba cd=dc\tef=fe    ", true, map[string]string{"ab": "ba", "cd": "dc", "ef": "fe"}},
		{"quotes", `a=b b="hello world"`, true, map[string]string{"a": "b", "b": "hello world"}},
		{"quotes2", `a="b c" b="hello world"`, true, map[string]string{"a": "b c", "b": "hello world"}},
		{"quotes contain quote", `a="b \""`, true, map[string]string{"a": `b "`}},
		{"quotes contain backslashes", `a="b \\"`, true, map[string]string{"a": `b \\`}},
		{"missing quotes", `a=b b=hello world`, false, nil},
		{"missing end quote", `a="b `, false, nil},
		{"quotes missing space", `a="b"b="hello world"`, false, nil},
		{"escape1", `a=b b=hello\ \ world`, true, map[string]string{"a": "b", "b": "hello  world"}},
		{"escape2", `a=b b=hello\\world`, true, map[string]string{"a": "b", "b": `hello\\world`}},
		{"escape3", `a=b b=hello\	world`, true, map[string]string{"a": "b", "b": `hello	world`}},
		{"bad escape", `a=b b=hello\  world`, false, nil},
		{"bad escape2", `a=b b=hello\\ world`, false, nil},
		{"valid key chars", `1aA_.c-d=val`, true, map[string]string{"1aA_.c-d": "val"}},
		{"invalid key char", `ab!=val`, false, nil},
		{"missing key", `=val`, false, nil},
		{"missing val", `key=`, false, nil},
		{"missing val2", `key=\`, false, nil},
		{"valid val chars", `key=\ \"val123.-_'!~#$%#*))*\"\ `, true, map[string]string{"key": ` "val123.-_'!~#$%#*))*" `}},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			result, err := parseKeyVals(test.input)
			if test.succeed {
				require.NoError(err)
				require.Equal(test.output, result)
			} else {
				require.Error(err)
			}
		})
	}
}
