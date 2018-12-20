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

package cache

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFSStore(t *testing.T) {
	t.Run("get_no_exist", func(t *testing.T) {
		require := require.New(t)

		tempDir, err := ioutil.TempDir("/tmp", "")
		require.NoError(err)
		defer os.RemoveAll(tempDir)
		tempFile, err := ioutil.TempFile(tempDir, "cache")
		require.NoError(err)

		d, err := time.ParseDuration("10s")
		require.NoError(err)
		store, err := NewFSStore(tempFile.Name(), tempDir, d)
		require.NoError(err)
		defer store.Cleanup()

		loc, err := store.Get("a")
		require.NoError(err)
		require.Equal("", loc)
	})

	t.Run("set_then_get", func(t *testing.T) {
		require := require.New(t)

		tempDir, err := ioutil.TempDir("/tmp", "")
		require.NoError(err)
		defer os.RemoveAll(tempDir)
		tempFile, err := ioutil.TempFile(tempDir, "cache")
		require.NoError(err)

		d, err := time.ParseDuration("10s")
		require.NoError(err)
		store, err := NewFSStore(tempFile.Name(), tempDir, d)
		require.NoError(err)
		defer store.Cleanup()

		err = store.Put("a", "b")
		require.NoError(err)
		value, err := store.Get("a")
		require.NoError(err)
		require.Equal("b", value)
	})
}
