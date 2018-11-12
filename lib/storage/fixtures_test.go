package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStoreFixture(t *testing.T) {
	require := require.New(t)

	store, cleanup := StoreFixture()
	require.NotNil(store)
	require.NoError(CleanupSandbox(store.RootDir))
	defer cleanup()
}

func TestStoreFixtureWithSampleImage(t *testing.T) {
	require := require.New(t)

	store, cleanup := StoreFixtureWithSampleImage()
	require.NotNil(store)
	require.NoError(CleanupSandbox(store.RootDir))
	defer cleanup()
}
