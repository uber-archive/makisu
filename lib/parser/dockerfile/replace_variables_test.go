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

func TestReplaceVariable(t *testing.T) {
	tests := []struct {
		desc       string
		key        string
		defaultVal string
		defaultCmd rune
		expected   string
		succeed    bool
		ok         bool
		vars       map[string]string
	}{
		{"simple", "key", "", 0, "VAL", true, true, map[string]string{"key": "VAL"}},
		{"missing val", "key", "", 0, "", true, false, nil},
		{"default - present", "key", "default", '-', "VAL", true, true, map[string]string{"key": "VAL"}},
		{"default - missing", "key", "default", '-', "default", true, true, nil},
		{"default + present", "key", "default", '+', "default", true, true, map[string]string{"key": "VAL"}},
		{"default + missing", "key", "", '+', "", false, false, nil},
		{"bad default command", "key", "default", 'z', "", false, false, nil},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)

			base := &replaceVarsBase{"", test.vars, test.key, nil, test.defaultCmd, test.defaultVal, false}
			val, ok, err := base.resolveCurrVar()
			if test.succeed {
				require.NoError(err)
			} else {
				require.Error(err)
			}
			require.Equal(test.ok, ok)
			require.Equal(test.expected, val)
		})
	}
}

func TestReplaceVariables(t *testing.T) {
	m := map[string]string{
		"key": "VAL", "VAL": "VAL2", "test_VAL": "VAL3",
		"VAL_test": "VAL4", "VAL2": "VAL5"}
	tests := []struct {
		desc     string
		input    string
		vars     map[string]string
		expected string
		succeed  bool
	}{
		{"no bracket", "text$key", m, "textVAL", true},
		{"back to back", "$key$key", m, "VALVAL", true},
		{"bracket", "text${key}", m, "textVAL", true},
		{"bracket unexpected end", "text${", nil, "", false},
		{"quotes", "text\"$key\"", m, "text\"VAL\"", true},
		{"recursive", "text${$key}", m, "textVAL2", true},
		{"recursive 2", "text${${key}}", m, "textVAL2", true},
		{"recursive 3", "text${test_$key}", m, "textVAL3", true},
		{"recursive 4", "text${${key}_test}", m, "textVAL4", true},
		{"missing key", "text$", nil, "text$", true},
		{"missing key bracket", "text${}", nil, "text${}", true},
		{"missing val", "text$key", nil, "text$key", true},
		{"missing val bracket", "text${key}", nil, "text${key}", true},
		{"missing recursive", "text${$VAL2}", m, "text${VAL5}", true},
		{"missing recursive 2", "text${${VAL2}}", m, "text${VAL5}", true},
		{"space following", "$key text", m, "VAL text", true},
		{"space following 2", "${key} text", m, "VAL text", true},
		{"no space following", "${key}text", m, "VALtext", true},
		{"spaces padded", "text $key text", m, "text VAL text", true},
		{"spaces padded bracket", "text ${key} text", m, "text VAL text", true},
		{"no spaces bracket", "text${key}text", m, "textVALtext", true},
		{"- default present", "text ${key:-default} text", m, "text VAL text", true},
		{"- default missing", "text ${key:-default} text", nil, "text default text", true},
		{"+ default present", "text ${key:+default} text", m, "text default text", true},
		{"+ default missing", "text ${key:+default} text", nil, "text  text", true},
		{"default recursive", "text ${$VAL:-default} text", m, "text VAL5 text", true},
		{"default recursive 2", "text ${${key}:-default} text", m, "text VAL2 text", true},
		{"default bad recursive", "text ${${$VAL:-default} text", nil, "", false},
		{"- default backslashes", `text ${key:-\\} text`, nil, `text \\ text`, true},
		{"- default bracket", `text ${key:-\}} text`, nil, "text } text", true},
		{"default recursive bracket", "text ${${VAL}:-default} text", m, "text VAL5 text", true},
		{"default unexpected end", "text ${key:", nil, "", false},
		{"default unexpected end 2", "text ${:", nil, "", false},
		{"bad default cmd", "text ${key:!default} text", nil, "", false},
		{"missing default key", "text ${:-} text", nil, "", false},
		{"missing default val", "text ${key:-} text", nil, "", false},
		{"no brackets valid", "/path/$key/", m, "/path/VAL/", true},
		{"no brackets invalid", "path-$key-", m, "path-$key-", true},
		{"escaped 1", "$key \\$key$key", m, "VAL $keyVAL", true},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)

			output, err := replaceVariables(test.input, test.vars)
			if test.succeed {
				require.NoError(err)
			} else {
				require.Error(err)
			}
			require.Equal(test.expected, output)
		})
	}
}
