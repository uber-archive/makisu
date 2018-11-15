//  Copyright (c) 2018 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
