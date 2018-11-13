package dockerfile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewEnvDirective(t *testing.T) {
	buildState := newParsingState(make(map[string]string))
	buildState.stageVars = map[string]string{"prefix": "test_", "suffix": "_test", "space": " "}

	tests := []struct {
		desc    string
		succeed bool
		input   string
		envs    map[string]string
	}{
		{"single", true, "env k1 v1", map[string]string{"k1": "v1"}},
		{"missing space", false, "env k1\tv1", nil},
		{"single spaces", true, "env k1 v1${space}v2 v3v4", map[string]string{"k1": "v1 v2 v3v4"}},
		{"single key-value", true, "env k1=v1", map[string]string{"k1": "v1"}},
		{"quotes", true, `env k1="v1a v1b"`, map[string]string{"k1": "v1a v1b"}},
		{"mutiple", true, "env k1=v1 k2=v2", map[string]string{"k1": "v1", "k2": "v2"}},
		{"substitution", true, "env k1=${prefix}v1 k2=v2$suffix", map[string]string{"k1": "test_v1", "k2": "v2_test"}},
		{"bad substitution", false, "env k1=${prefixv1 k2=v2$suffix", nil},
		{"quotes_substitution", true, `env k1="v1a${space}v1b"`, map[string]string{"k1": "v1a v1b"}},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			directive, err := newDirective(test.input, buildState)
			if test.succeed {
				require.NoError(err)
				env, ok := directive.(*EnvDirective)
				require.True(ok)
				require.Equal(test.envs, env.Envs)
			} else {
				require.Error(err)
			}
		})
	}
}
