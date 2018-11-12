package metadata

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateFromSuffix(t *testing.T) {
	require := require.New(t)

	lat := CreateFromSuffix(_lastAccessTimeSuffix)
	require.Equal(_lastAccessTimeSuffix, lat.GetSuffix())
}

func TestCreateFromSuffixFail(t *testing.T) {
	require := require.New(t)

	require.Nil(CreateFromSuffix(""))
}
