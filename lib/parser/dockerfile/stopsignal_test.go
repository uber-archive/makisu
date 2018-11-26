package dockerfile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewStopsignalDirective(t *testing.T) {
	buildState := newParsingState(make(map[string]string))

	tests := []struct {
		desc    string
		succeed bool
		input   string
		signal  int
	}{
		{"simple", true, "stopsignal 9", 9},
		{"not int", false, "stopsignal 123asd", 0},
		{"bad signal", false, "stopsignal -1", 0},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			directive, err := newDirective(test.input, buildState)
			if test.succeed {
				require.NoError(err)
				casted, ok := directive.(*StopsignalDirective)
				require.True(ok)
				require.Equal(test.signal, casted.Signal)
			} else {
				require.Error(err)
			}
		})
	}
}
