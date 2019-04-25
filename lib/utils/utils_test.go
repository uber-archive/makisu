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

package utils

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uber/makisu/lib/utils/testutil"
)

func TestMultiError(t *testing.T) {
	require := require.New(t)

	me := NewMultiErrors()
	require.Nil(me.Collect())

	firstError := fmt.Errorf("first error")
	me.Add(firstError)
	require.Equal(firstError, me.Collect())

	var wg sync.WaitGroup
	var mu sync.Mutex
	errStr := firstError.Error()
	for i := 0; i < 100; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			time.Sleep(1 * time.Millisecond)
			err := fmt.Errorf("error %d", i)
			me.Add(err)
			mu.Lock()
			defer mu.Unlock()
			errStr = fmt.Sprintf("%s; %s", errStr, err.Error())
		}()
	}

	wg.Wait()
	require.NotNil(me.Collect())
	collectedErrors := strings.Split(me.Collect().Error(), ";")
	sort.Strings(collectedErrors)
	expectedErrors := strings.Split(errStr, ";")
	sort.Strings(expectedErrors)
	require.Equal(expectedErrors, collectedErrors)
}

func TestDefaultEnv(t *testing.T) {
	require := require.New(t)

	require.Equal("hello", DefaultEnv("DOES NOT EXIST", "hello"))

	err := os.Setenv("DOES NOT EXIST", "hello2")
	require.NoError(err)
	require.Equal("hello2", DefaultEnv("DOES NOT EXIST", "hello"))
}

func TestMust(t *testing.T) {
	Must(true, "Should not exit and fail the tests")
}

func TestConvertStringSlice(t *testing.T) {
	require := require.New(t)

	values := []string{"a=b", "c=d", "f"}
	out := ConvertStringSliceToMap(values)
	require.NotNil(out)
	require.Equal("b", out["a"])
	require.Equal("d", out["c"])
	require.Equal("", out["f"])
}

func TestMergeEnv(t *testing.T) {
	require := require.New(t)

	env1 := []string{"a=b", "c=d"}
	env2 := map[string]string{"a": "e", "g": "h"}
	out := MergeEnv(env1, env2)
	require.NotNil(out)
	require.Contains(out, "a=e")
	require.Contains(out, "c=d")
	require.Contains(out, "g=h")

	out = MergeEnv(env1, nil)
	require.NotNil(out)
	require.Contains(out, "a=b")
	require.Contains(out, "c=d")

	out = MergeEnv(nil, env2)
	require.NotNil(out)
	require.Contains(out, "a=e")
	require.Contains(out, "g=h")
}

func TestMergeStringMaps(t *testing.T) {
	require := require.New(t)

	map1 := map[string]string{"a": "b", "c": "d"}
	map2 := map[string]string{"a": "e", "g": "h"}

	out := MergeStringMaps(map1, map2)
	require.NotNil(out)
	require.Equal("e", out["a"])
	require.Equal("d", out["c"])
	require.Equal("h", out["g"])

	out = MergeStringMaps(map1, nil)
	require.NotNil(out)
	require.Equal("b", out["a"])
	require.Equal("d", out["c"])

	out = MergeStringMaps(nil, map2)
	require.NotNil(out)
	require.Equal("e", out["a"])
	require.Equal("h", out["g"])
}

func TestMergeStructMaps(t *testing.T) {
	require := require.New(t)

	map1 := map[string]struct{}{"a": {}}
	map2 := map[string]struct{}{"b": {}}

	out := MergeStructMaps(map1, map2)
	require.NotNil(out)
	require.Contains(out, "a")
	require.Contains(out, "b")
}

func TestMin(t *testing.T) {
	require := require.New(t)

	result := Min(1, 2, 3, 4, 5)
	require.Equal(int64(1), result)

	result = Min(-1, -2, 3, 4, 5)
	require.Equal(int64(-2), result)

	result = Min(1, 2, 3, 4, -10)
	require.Equal(int64(-10), result)
}

func TestIsSpecialFileWithSocket(t *testing.T) {
	require := require.New(t)
	tmpDir, err := ioutil.TempDir("/tmp", "makisu-test")
	require.NoError(err)
	defer os.RemoveAll(tmpDir)

	socketPath := filepath.Join(tmpDir, "test.sock")
	_, err = net.Listen("unix", socketPath)
	require.NoError(err)

	fi, err := os.Stat(socketPath)
	require.NoError(err)
	require.True(IsSpecialFile(fi))
}

func TestResolveChown(t *testing.T) {
	tests := []struct {
		desc    string
		succeed bool
		chown   string
		uid     int
		gid     int
	}{
		{"missing group", false, "user:", 0, 0},
		{"missing user", false, ":group", 0, 0},
		{"missing group or user", false, ":", 0, 0},
		{"empty", true, "", 0, 0},
		{"uid no group", true, "1", 1, 1},
		{"uid and gid", true, "1:2", 1, 2},
		{"user no group", true, fmt.Sprintf("%s", testutil.CurrUser()), testutil.CurrUID(), testutil.CurrUID()},
		{"user and gid", true, fmt.Sprintf("%s:1", testutil.CurrUser()), testutil.CurrUID(), 1},
		{"uid and gid", true, fmt.Sprintf("%d:%d", testutil.CurrUID(), testutil.CurrGID()), testutil.CurrUID(), testutil.CurrGID()},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			uid, gid, err := ResolveChown(test.chown)
			if test.succeed {
				require.Equal(test.uid, uid)
				require.Equal(test.gid, gid)
			} else {
				require.Error(err)
			}
		})
	}
}
