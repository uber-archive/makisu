package archive

import (
	"archive/tar"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateHeader(t *testing.T) {
	t.Run("Directory", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		l := newMemLayer()
		src, err := ioutil.TempDir(tmpRoot, "test")
		require.NoError(err)
		fi, err := os.Lstat(src)
		require.NoError(err)

		hdr, err := l.createHeader(tmpRoot, src, "/tmp/testDest", fi)
		require.NoError(err)

		require.Equal("tmp/testDest/", hdr.Name)
		require.Equal(tar.TypeDir, int32(hdr.Typeflag))
	})

	t.Run("RegularFile", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		l := newMemLayer()
		src, err := ioutil.TempFile(tmpRoot, "test")
		require.NoError(err)
		fi, err := os.Lstat(src.Name())
		require.NoError(err)

		hdr, err := l.createHeader(tmpRoot, src.Name(), "/tmp/testDest", fi)
		require.NoError(err)

		require.Equal("tmp/testDest", hdr.Name)
		require.Equal(tar.TypeReg, int32(hdr.Typeflag))
	})

	t.Run("Symlink", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		l := newMemLayer()
		tmp, err := ioutil.TempFile(tmpRoot, "test")
		require.NoError(err)
		link := filepath.Join(tmpRoot, "link")
		require.NoError(os.Symlink(tmp.Name(), link))
		fi, err := os.Lstat(link)
		require.NoError(err)

		hdr, err := l.createHeader(tmpRoot, link, "/tmp/testDest", fi)
		require.NoError(err)

		require.Equal("tmp/testDest", hdr.Name)
		require.Equal(tar.TypeSymlink, int32(hdr.Typeflag))
	})
}

func TestAddHeader(t *testing.T) {
	t.Run("RegularFile", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		l := newMemLayer()
		src, err := ioutil.TempFile(tmpRoot, "test")
		require.NoError(err)
		fi, err := os.Lstat(src.Name())
		require.NoError(err)
		dst := "/tmp/testDest"

		hdr, err := l.createHeader(tmpRoot, src.Name(), dst, fi)
		require.NoError(err)

		added := l.addHeader(src.Name(), dst, hdr)
		memfile, ok := l.files[dst]
		require.True(ok)
		require.Equal(added, memfile)
		contentFile, ok := memfile.(*contentMemFile)
		require.True(ok)
		require.Equal(src.Name(), contentFile.src)
		require.Equal(dst, contentFile.dst)
		require.Equal(hdr, contentFile.hdr)
	})

	t.Run("Whiteout", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		l := newMemLayer()
		src, err := ioutil.TempFile(tmpRoot, "test")
		require.NoError(err)
		fi, err := os.Lstat(src.Name())
		require.NoError(err)
		dst := "/tmp/.wh.testDest"
		del := "/tmp/testDest"

		hdr, err := l.createHeader(tmpRoot, src.Name(), dst, fi)
		require.NoError(err)

		added := l.addHeader(src.Name(), dst, hdr)
		memfile, ok := l.files[del]
		require.True(ok)
		require.Equal(added, memfile)
		whiteout, ok := memfile.(*whiteoutMemFile)
		require.True(ok)
		require.Equal(del, whiteout.del)
		require.Equal("tmp/.wh.testDest", whiteout.hdr.Name)
	})
}

func TestAddWhiteout(t *testing.T) {
	t.Run("RegularFile", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		l := newMemLayer()
		del := "/tmp/testDest"

		added, err := l.addWhiteout(del)
		require.NoError(err)
		memfile, ok := l.files[del]
		require.True(ok)
		require.Equal(added, memfile)
		whiteout, ok := memfile.(*whiteoutMemFile)
		require.True(ok)
		require.Equal(del, whiteout.del)
		require.Equal("tmp/.wh.testDest", whiteout.hdr.Name)
	})

	t.Run("RejectWhiteout", func(t *testing.T) {
		require := require.New(t)

		tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
		require.NoError(err)
		defer os.RemoveAll(tmpRoot)

		l := newMemLayer()
		del := "/tmp/.wh.testDest"

		_, err = l.addWhiteout(del)
		require.Error(err)
	})
}
