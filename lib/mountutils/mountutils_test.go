package mountutils

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsMountpoint(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "mountpoint")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(`cgroup /etc/hostname etx4 ro,nosuid,nodev,noexec,mode=755 0 0
cgroup /etc/hosts etx4 ro,nosuid,nodev,noexec,mode=755 0 0
`))
	require.NoError(t, err)

	info := newMountInfo()
	info.mountsFile = tmpfile.Name()

	isMount, err := info.isMountpoint("/etc/hosts")
	require.NoError(t, err)
	require.True(t, isMount)

	isMount, err = info.isMountpoint("/etc/hosts.txt")
	require.NoError(t, err)
	require.False(t, isMount)
}

func TestIsMounted(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "mountpoint")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(`cgroup /etc/hostname etx4 ro,nosuid,nodev,noexec,mode=755 0 0
cgroup /etc/hosts etx4 ro,nosuid,nodev,noexec,mode=755 0 0
cgroup /var/cache etx4 ro,nosuid,nodev,noexec,mode=755 0 0
`))
	require.NoError(t, err)

	info := newMountInfo()
	info.mountsFile = tmpfile.Name()

	isMounted, err := info.isMounted("/var/cached")
	require.NoError(t, err)
	require.False(t, isMounted)

	isMounted, err = info.isMounted("/var/cache")
	require.NoError(t, err)
	require.True(t, isMounted)

	isMounted, err = info.isMounted("/var/cache/stuff")
	require.NoError(t, err)
	require.True(t, isMounted)
}

func TestContainsMountpoint(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "mountpoint")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(`cgroup /etc/hostname etx4 ro,nosuid,nodev,noexec,mode=755 0 0
cgroup /etc/hosts etx4 ro,nosuid,nodev,noexec,mode=755 0 0
cgroup /var/cache etx4 ro,nosuid,nodev,noexec,mode=755 0 0
cgroup /var/run/some/directory etx4 ro,nosuid,nodev,noexec,mode=755 0 0
`))
	require.NoError(t, err)

	info := newMountInfo()
	info.mountsFile = tmpfile.Name()

	contains, err := info.containsMountpoint("/var/run")
	require.NoError(t, err)
	require.True(t, contains)

	contains, err = info.containsMountpoint("/var/")
	require.NoError(t, err)
	require.True(t, contains)

	contains, err = info.containsMountpoint("/var/other")
	require.NoError(t, err)
	require.False(t, contains)
}

func TestInitFailure(t *testing.T) {
	t.Run("Bad /proc/mounts format", func(t *testing.T) {
		tmpfile, err := ioutil.TempFile("", "mountpoint")
		require.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.Write([]byte(`badline`))
		require.NoError(t, err)

		info := newMountInfo()
		info.mountsFile = tmpfile.Name()

		_, err = info.isMountpoint("/etc/hosts")
		require.Error(t, err)
	})

	t.Run("No /proc/mounts file on disk", func(t *testing.T) {
		info := newMountInfo()
		info.mountsFile = "file_that_does_not_exist"

		isMount, err := info.isMountpoint("/etc/hosts")
		require.NoError(t, err)
		require.False(t, isMount)
	})
}
