package dockerfile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRunDirective(t *testing.T) {
	buildState := newParsingState(make(map[string]string))
	buildState.stageVars = map[string]string{"prefix": "test_", "suffix": "_test", "comma": ","}

	tests := []struct {
		desc    string
		succeed bool
		input   string
		cmd     string
	}{
		{"good json", true, `run ["this", "cmd"]`, "this cmd"},
		{"substitution", true, `run ["${prefix}this", "cmd${suffix}"]`, "test_this cmd_test"},
		{"substitution2", true, `run ["this"$comma "cmd"]`, "this cmd"},
		{"bad substitution", false, `run ["${prefixthis", "cmd${suffix}"]`, ""},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			directive, err := newDirective(test.input, buildState)
			if test.succeed {
				require.NoError(err)
				run, ok := directive.(*RunDirective)
				require.True(ok)
				require.Equal(test.cmd, run.Cmd)
			} else {
				require.Error(err)
			}
		})
	}
}
