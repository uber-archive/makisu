package fileio

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConcatDirectoryContents(t *testing.T) {
	t.Run("no files", func(t *testing.T) {
		dir, err := ioutil.TempDir("", "makisu-test")
		require.NoError(t, err)
		defer os.RemoveAll(dir)

		contents, err := ConcatDirectoryContents(dir)
		require.NoError(t, err)
		require.Len(t, contents, 0)
	})

	t.Run("some files", func(t *testing.T) {
		dir, err := ioutil.TempDir("", "makisu-test")
		require.NoError(t, err)
		defer os.RemoveAll(dir)

		f1 := filepath.Join(dir, "f1")
		err = ioutil.WriteFile(f1, []byte("TEST1"), os.ModePerm)
		require.NoError(t, err)

		f2 := filepath.Join(dir, "f2")
		err = ioutil.WriteFile(f2, []byte("TEST2"), os.ModePerm)
		require.NoError(t, err)

		contents, err := ConcatDirectoryContents(dir)
		require.NoError(t, err)
		require.Equal(t, "TEST1TEST2", string(contents))
	})
}
