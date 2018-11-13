package dockerfile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitArgs(t *testing.T) {
	tests := []struct {
		desc    string
		input   string
		succeed bool
		output  []string
	}{
		{"single", "  a	", true, []string{"a"}},
		{"single quoted", `  "a   "	`, true, []string{"a   "}},
		{"quoted contains backslash", `  "a \  "	`, true, []string{`a \  `}},
		{"quoted contains quote", `  "a \"  "	`, true, []string{`a "  `}},
		{"single quoted incomplete", `"a `, false, nil},
		{"multiple", `a=b   c d  "e f" \"g`, true, []string{"a=b", "c", "d", "e f", "\"g"}},
		{"arg chars", "`1'{}$@#! *&@)(*_&", true, []string{"`1'{}$@#!", "*&@)(*_&"}},
		{"non-escapes", "\\`1'{}$\\@#! *&\\;)\\(*_&", true, []string{"\\`1'{}$\\@#!", "*&\\;)\\(*_&"}},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			result, err := splitArgs(test.input)
			if test.succeed {
				require.NoError(err)
				require.Equal(test.output, result)
			} else {
				require.Error(err)
			}
		})
	}
}
