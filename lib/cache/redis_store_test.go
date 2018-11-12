package cache

import (
	"testing"

	"github.com/alicebob/miniredis"
	"github.com/stretchr/testify/require"
)

func TestRedisStore(t *testing.T) {
	t.Run("get_no_exist", func(t *testing.T) {
		s, err := miniredis.Run()
		if err != nil {
			panic(err)
		}
		defer s.Close()

		store, err := NewRedisStore(s.Addr(), 10)
		require.NoError(t, err)

		loc, err := store.Get("a")
		require.NoError(t, err)
		require.Equal(t, "", loc)
	})
	t.Run("set_then_get", func(t *testing.T) {
		s, err := miniredis.Run()
		if err != nil {
			panic(err)
		}
		defer s.Close()

		store, err := NewRedisStore(s.Addr(), 10)
		require.NoError(t, err)

		defer store.Cleanup()
		err = store.Put("a", "b")
		require.NoError(t, err)
		loc, err := store.Get("a")
		require.NoError(t, err)
		require.Equal(t, "b", loc)
	})
}
