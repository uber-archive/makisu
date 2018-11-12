package cache

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFSStore(t *testing.T) {
	t.Run("get_no_exist", func(t *testing.T) {
		tmpdir, err := ioutil.TempDir("/tmp", "")
		require.NoError(t, err)
		defer os.RemoveAll(tmpdir)
		store := NewFSStore(tmpdir)
		defer store.Cleanup()
		loc, err := store.Get("a")
		require.NoError(t, err)
		require.Equal(t, "", loc)
	})
	t.Run("set_then_get", func(t *testing.T) {
		tmpdir, err := ioutil.TempDir("/tmp", "")
		require.NoError(t, err)
		defer os.RemoveAll(tmpdir)
		store := NewFSStore(tmpdir)
		defer store.Cleanup()
		err = store.Put("a", "b")
		require.NoError(t, err)
		loc, err := store.Get("a")
		require.NoError(t, err)
		require.Equal(t, "b", loc)
	})
}
