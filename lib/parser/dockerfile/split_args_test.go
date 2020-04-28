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

func TestSplitArgs(t *testing.T) {
	tests := []struct {
		desc       string
		input      string
		keepQuotes bool
		succeed    bool
		output     []string
	}{
		{"single", "  a	", false, true, []string{"a"}},
		{"single quoted", `  "a   "	`, false, true, []string{"a   "}},
		{"quoted contains backslash", `  "a \  "	`, false, true, []string{`a \  `}},
		{"quoted contains quote", `  "a \"  "	`, false, true, []string{`a "  `}},
		{"single quoted incomplete", `"a `, false, false, nil},
		{"multiple", `a=b   c d  "e f" \"g`, false, true, []string{"a=b", "c", "d", "e f", "\"g"}},
		{"arg chars", "`1'{}$@#! *&@)(*_&", false, true, []string{"`1'{}$@#!", "*&@)(*_&"}},
		{"non-escapes", "\\`1'{}$\\@#! *&\\;)\\(*_&", false, true, []string{"\\`1'{}$\\@#!", "*&\\;)\\(*_&"}},
		{"keep quotes", `echo "${MESSAGE}" "multi \"word"`, true, true, []string{"echo", `"${MESSAGE}"`, `"multi \"word"`}},
		{"keep quotes - inclusion", `echo "single argument \\\"keepQuotes\\\""`, true, true, []string{"echo", `"single argument \\\"keepQuotes\\\""`}},
		{"keep quotes - shell like", `if true; then echo "you are just here for the 0 exit code"; else exit 1; fi`, true, true, []string{"if", "true", ";", "then", "echo", `"you are just here for the 0 exit code"`, ";", "else", "exit", "1", ";", "fi"}},
		{"keep quotes - parse or and and correctly", `echo "toto"&&echo "tata"||echo "test"`, true, true, []string{"echo", `"toto"`, "&&", "echo", `"tata"`, "||", "echo", `"test"`}},
		{"keep quotes - harder and and or", `echo "more space" &&echo "detached after"&&echo "toto"& `, true, true, []string{"echo", "\"more space\"", "&&", "echo", "\"detached after\"", "&&", "echo", "\"toto\"", "&"}},
		{"keep quotes - one and", `echo "more space"& `, true, true, []string{"echo", "\"more space\"", "&"}},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			result, err := splitArgs(test.input, test.keepQuotes)
			if test.succeed {
				require.NoError(err)
				require.Equal(test.output, result)
			} else {
				require.Error(err)
			}
		})
	}
}
