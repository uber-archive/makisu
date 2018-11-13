package dockerfile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewVolumeDirective(t *testing.T) {
	buildState := newParsingState(make(map[string]string))
	buildState.stageVars = map[string]string{"prefix": "test_", "suffix": "_test", "comma": ","}

	tests := []struct {
		desc    string
		succeed bool
		input   string
		volumes []string
	}{
		{"shell single", true, "volume v1", []string{"v1"}},
		{"shell multi", true, "volume v1 v2", []string{"v1", "v2"}},
		{"shell substitution", true, "volume ${prefix}v1 v2$suffix", []string{"test_v1", "v2_test"}},
		{"json single", true, `volume ["v1"]`, []string{"v1"}},
		{"json multi", true, `volume ["v1", "v2"]`, []string{"v1", "v2"}},
		{"json substitution", true, `volume ["${prefix}v1"$comma "v2${suffix}"]`, []string{"test_v1", "v2_test"}},
		{"bad substitution", false, "volume ${prefixv1 v2$suffix", nil},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			directive, err := newDirective(test.input, buildState)
			if test.succeed {
				require.NoError(err)
				volume, ok := directive.(*VolumeDirective)
				require.True(ok)
				require.Equal(test.volumes, volume.Volumes)
			} else {
				require.Error(err)
			}
		})
	}
}
