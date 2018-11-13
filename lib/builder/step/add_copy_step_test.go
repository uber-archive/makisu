package step

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContextDirs(t *testing.T) {
	require := require.New(t)

	srcs := []string{}
	ac, err := newAddCopyStep(Copy, "", "", "", srcs, "", false)
	require.NoError(err)
	stage, paths := ac.ContextDirs()
	require.Equal("", stage)
	require.Len(paths, 0)

	srcs = []string{"src"}
	ac, err = newAddCopyStep(Copy, "", "", "", srcs, "", false)
	require.NoError(err)
	stage, paths = ac.ContextDirs()
	require.Equal("", stage)
	require.Len(paths, 0)

	srcs = []string{"src"}
	ac, err = newAddCopyStep(Copy, "", "", "stage", srcs, "", false)
	require.NoError(err)
	stage, paths = ac.ContextDirs()
	require.Equal("stage", stage)
	require.Len(paths, 1)
}
