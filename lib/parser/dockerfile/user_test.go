package dockerfile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewUserDirective(t *testing.T) {
	buildState := newParsingState(make(map[string]string))
	buildState.stageVars = map[string]string{"prefix": "test_", "suffix": "_test", "colon": ":"}

	tests := []struct {
		desc    string
		succeed bool
		input   string
		user    string
	}{
		{"too many args", false, "user u1:g1 u2:g2", ""},
		{"both", true, "user u1:g1", "u1:g1"},
		{"substitution", true, "user ${prefix}u1${colon}g1$suffix", "test_u1:g1_test"},
		{"bad substitution", false, "user ${prefix}u1${colong1$suffix", ""},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			directive, err := newDirective(test.input, buildState)
			if test.succeed {
				require.NoError(err)
				user, ok := directive.(*UserDirective)
				require.True(ok)
				require.Equal(test.user, user.User)
			} else {
				require.Error(err)
			}
		})
	}
}
