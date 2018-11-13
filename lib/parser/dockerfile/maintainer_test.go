package dockerfile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewMaintainerDirective(t *testing.T) {
	buildState := newParsingState(make(map[string]string))
	buildState.stageVars = map[string]string{"arg1": "foo", "arg2": "bar", "space": " "}

	tests := []struct {
		desc    string
		succeed bool
		input   string
		author  string
	}{
		{"simple", true, "maintainer foo bar", "foo bar"},
		{"special char", true, "maintainer test* <test@exmaple.com> &\"", "test* <test@exmaple.com> &\""},
		{"no substitution", true, "maintainer test ${arg1}${arg2} $space", "test ${arg1}${arg2} $space"},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			directive, err := newDirective(test.input, buildState)
			if test.succeed {
				require.NoError(err)
				expose, ok := directive.(*MaintainerDirective)
				require.True(ok)
				require.Equal(test.author, expose.author)
			} else {
				require.Error(err)
			}
		})
	}
}
