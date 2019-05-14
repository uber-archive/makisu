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

package storage

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootPreserver(t *testing.T) {
	require := require.New(t)

	// This is /
	tmpRoot, err := ioutil.TempDir("/tmp", "makisu-test")
	require.NoError(err)
	// defer os.RemoveAll(tmpRoot)

	// This is the storage directory
	storageDir, err := ioutil.TempDir(tmpRoot, "storage")
	require.NoError(err)

	initialRootBackup := filepath.Join(storageDir, rootPreserverBackupDir)

	_hello := []byte("hello")
	shouldBeCopied := "/test.txt"
	shouldNotBeCopied := "/test2.txt"
	require.NoError(ioutil.WriteFile(filepath.Join(tmpRoot, shouldBeCopied), _hello, os.ModePerm))
	require.NoError(ioutil.WriteFile(filepath.Join(tmpRoot, shouldNotBeCopied), _hello, os.ModePerm))

	// Before root preserver:
	// /tmp/makisu-test362631474/
	// /tmp/makisu-test362631474/test2.txt
	// /tmp/makisu-test362631474/storage054318825
	// /tmp/makisu-test362631474/test.txt

	rootPreserver, err := NewRootPreserver(tmpRoot, storageDir, []string{storageDir, filepath.Join(tmpRoot, shouldNotBeCopied)})
	require.NoError(err)

	// Now we want:
	// /tmp/makisu-test362631474/
	// /tmp/makisu-test362631474/test2.txt
	// /tmp/makisu-test362631474/storage054318825
	// /tmp/makisu-test362631474/storage054318825/test.txt
	// /tmp/makisu-test362631474/test.txt

	b, err := ioutil.ReadFile(filepath.Join(initialRootBackup, shouldBeCopied))
	require.NoError(err)
	require.Equal(_hello, b)

	b, err = ioutil.ReadFile(filepath.Join(initialRootBackup, shouldNotBeCopied))
	require.Error(err)

	// Remove initial file that would have been copied
	err = os.Remove(filepath.Join(tmpRoot, shouldBeCopied))
	require.NoError(err)

	err = rootPreserver.RestoreRoot()
	require.NoError(err)

	b, err = ioutil.ReadFile(filepath.Join(tmpRoot, shouldBeCopied))
	require.NoError(err)
	require.Equal(_hello, b)

	b, err = ioutil.ReadFile(filepath.Join(tmpRoot, shouldNotBeCopied))
	require.NoError(err)
	require.Equal(_hello, b)

	// Check that the backup dir is deleted
	if _, err := os.Stat(filepath.Join(storageDir, rootPreserverBackupDir)); !os.IsNotExist(err) {
		require.Failf("Storage dir: %s should not exists", storageDir)
	}
}
