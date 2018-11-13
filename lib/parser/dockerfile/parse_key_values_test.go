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
