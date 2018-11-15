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
