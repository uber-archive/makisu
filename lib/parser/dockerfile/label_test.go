package dockerfile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewLabelDirective(t *testing.T) {
	buildState := newParsingState(make(map[string]string))
	buildState.stageVars = map[string]string{"prefix": "test_", "suffix": "_test", "space": " "}

	tests := []struct {
		desc    string
		succeed bool
		input   string
		labels  map[string]string
	}{
		{"single", true, "label k1=v1", map[string]string{"k1": "v1"}},
		{"quotes", true, `label k1="v1a v1b"`, map[string]string{"k1": "v1a v1b"}},
		{"mutiple", true, "label k1=v1 k2=v2", map[string]string{"k1": "v1", "k2": "v2"}},
		{"bad pair", false, "label k1 v1", nil},
		{"substitution", true, "label k1=${prefix}v1 k2=v2$suffix", map[string]string{"k1": "test_v1", "k2": "v2_test"}},
		{"bad substitution", false, "label k1=${prefix}v1 k2=v2${suffix", nil},
		{"quotes_substitution", true, `label k1="v1a${space}v1b"`, map[string]string{"k1": "v1a v1b"}},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			directive, err := newDirective(test.input, buildState)
			if test.succeed {
				require.NoError(err)
				label, ok := directive.(*LabelDirective)
				require.True(ok)
				require.Equal(test.labels, label.Labels)
			} else {
				require.Error(err)
			}
		})
	}
}
