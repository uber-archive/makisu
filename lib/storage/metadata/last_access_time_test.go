package metadata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLastAccessTimeFactory(t *testing.T) {
	require := require.New(t)

	lat := lastAccessTimeFactory{}.Create(_lastAccessTimeSuffix)
	require.Equal(_lastAccessTimeSuffix, lat.GetSuffix())
}

func TestLastAccessTimeMovable(t *testing.T) {
	require := require.New(t)

	lat := NewLastAccessTime(time.Now().Add(-time.Hour))
	require.True(lat.Movable())
}

func TestLastAccessTimeSerialization(t *testing.T) {
	require := require.New(t)

	lat := NewLastAccessTime(time.Now().Add(-time.Hour))
	b, err := lat.Serialize()
	require.NoError(err)

	var newLat LastAccessTime
	require.NoError(newLat.Deserialize(b))
	require.Equal(lat.Time.Unix(), newLat.Time.Unix())
}
