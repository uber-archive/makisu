package dockerfile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewWorkdirDirective(t *testing.T) {
	buildState := newParsingState(make(map[string]string))
	buildState.stageVars = map[string]string{"prefix": "test_", "suffix": "_test"}

	tests := []struct {
		desc    string
		succeed bool
		input   string
		dir     string
	}{
		{"plain", true, "workdir /a/b", "/a/b"},
		{"substitution", true, "workdir /${prefix}a/b$suffix", "/test_a/b_test"},
		{"bad substitution", false, "workdir /${prefixa/b$suffix", ""},
		{"extra arg", false, "workdir /${prefixa}/b$suffix /root", ""},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			directive, err := newDirective(test.input, buildState)
			if test.succeed {
				require.NoError(err)
				workdir, ok := directive.(*WorkdirDirective)
				require.True(ok)
				require.Equal(test.dir, workdir.WorkingDir)
			} else {
				require.Error(err)
			}
		})
	}
}
