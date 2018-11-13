package dockerfile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewArgDirective(t *testing.T) {
	buildState := newParsingState(make(map[string]string))
	buildState.stageVars = map[string]string{"prefix": "test_", "suffix": "_test", "space": " "}

	tests := []struct {
		desc       string
		succeed    bool
		input      string
		name       string
		defaultVal string
	}{
		{"no default", true, "arg k1", "k1", ""},
		{"default", true, "arg k1=v1", "k1", "v1"},
		{"multiple kv pairs", false, "arg k1=v1 k2=v2", "", ""},
		{"multiple single args", false, "arg k1 k2", "", ""},
		{"bad args", false, `arg "k1"k2`, "", ""},
		{"quotes", true, `arg k1="v1a v1b"`, "k1", "v1a v1b"},
		{"quotes substitution", true, `arg ${prefix}k1${suffix}="${prefix}v1a${space}v1b${suffix}"`, "test_k1_test", "test_v1a v1b_test"},
		{"bad substitution", false, "arg ${prefix=val", "", ""},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			directive, err := newDirective(test.input, buildState)
			if test.succeed {
				require.NoError(err)
				arg, ok := directive.(*argDirective)
				require.True(ok)
				require.Equal(test.name, arg.name)
				require.Equal(test.defaultVal, arg.defaultVal)
			} else {
				require.Error(err)
			}
		})
	}
}
