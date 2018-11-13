package dockerfile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewCmdDirective(t *testing.T) {
	buildState := newParsingState(make(map[string]string))
	buildState.stageVars = map[string]string{"prefix": "test_", "suffix": "_test", "comma": ","}

	tests := []struct {
		desc    string
		succeed bool
		input   string
		cmd     []string
	}{
		{"good json", true, `cmd ["this", "cmd"]`, []string{"this", "cmd"}},
		{"substitution", true, `cmd ["${prefix}this", "cmd${suffix}"]`, []string{"test_this", "cmd_test"}},
		{"substitution 2", true, `cmd ["this"$comma "cmd"]`, []string{"this", "cmd"}},
		{"good cmd", true, "cmd this cmd", []string{"this", "cmd"}},
		{"quotes", true, `cmd "this cmd"`, []string{"this cmd"}},
		{"quotes 2", true, `cmd "this cmd" cmd2 "and cmd 3"`, []string{"this cmd", "cmd2", "and cmd 3"}},
		{"substitution", true, "cmd ${prefix}this cmd$suffix", []string{"test_this", "cmd_test"}},
		{"bad json", false, `cmd ["this, "cmd"]`, nil},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			directive, err := newDirective(test.input, buildState)
			if test.succeed {
				require.NoError(err)
				cmd, ok := directive.(*CmdDirective)
				require.True(ok)
				require.Equal(test.cmd, cmd.Cmd)
			} else {
				require.Error(err)
			}
		})
	}
}
