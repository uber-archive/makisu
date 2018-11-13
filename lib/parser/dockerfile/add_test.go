package dockerfile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewAddDirective(t *testing.T) {
	buildState := newParsingState(make(map[string]string))
	buildState.stageVars = map[string]string{"prefix": "test_", "suffix": "_test", "comma": ","}

	tests := []struct {
		desc    string
		succeed bool
		input   string
		srcs    []string
		dst     string
		chown   string
	}{
		{"shell single source", true, `add src dst`, []string{"src"}, "dst", ""},
		{"shell multi source", true, `add src1 src2 dst`, []string{"src1", "src2"}, "dst", ""},
		{"shell substitution", true, `add src1 ${prefix}src2 dst$suffix`, []string{"src1", "test_src2"}, "dst_test", ""},
		{"shell substitution bad", false, "add src1 ${prefix", nil, "", ""},
		{"shell chown", true, `add --chown=user:group src dst`, []string{"src"}, "dst", "user:group"},
		{"shell chown bad", false, `add --chown= src dst`, nil, "", ""},
		{"shell chown substitution", true, `add --chown=${prefix}user:group src1 ${prefix}src2 dst$suffix`, []string{"src1", "test_src2"}, "dst_test", "test_user:group"},
		{"json bad", false, `add ["src"]`, nil, "", ""},
		{"json single source", true, `add ["src", "dst"]`, []string{"src"}, "dst", ""},
		{"json multi source", true, `add ["src1", "src2", "dst"]`, []string{"src1", "src2"}, "dst", ""},
		{"json substitution", true, `add ["src1"$comma "src2${suffix}", "${prefix}dst"]`, []string{"src1", "src2_test"}, "test_dst", ""},
		{"json chown", true, `add --chown=user:group ["src", "dst"]`, []string{"src"}, "dst", "user:group"},
		{"json chown substitution", true, `add --chown=${prefix}user:group ["src1"$comma "src2${suffix}", "${prefix}dst"]`, []string{"src1", "src2_test"}, "test_dst", "test_user:group"},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			directive, err := newDirective(test.input, buildState)
			if test.succeed {
				require.NoError(err)
				cast, ok := directive.(*AddDirective)
				require.True(ok)
				require.Equal(test.srcs, cast.Srcs)
				require.Equal(test.dst, cast.Dst)
				require.Equal(test.chown, cast.Chown)
			} else {
				require.Error(err)
			}
		})
	}
}
