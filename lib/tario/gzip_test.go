package tario

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetCompressionLevelFail(t *testing.T) {
	require := require.New(t)

	require.Error(SetCompressionLevel("invalid"))
}
