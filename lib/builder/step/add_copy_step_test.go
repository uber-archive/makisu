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

package step

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContextDirs(t *testing.T) {
	require := require.New(t)

	srcs := []string{}
	ac, err := newAddCopyStep(Copy, "", "", "", srcs, "", false)
	require.NoError(err)
	stage, paths := ac.ContextDirs()
	require.Equal("", stage)
	require.Len(paths, 0)

	srcs = []string{"src"}
	ac, err = newAddCopyStep(Copy, "", "", "", srcs, "", false)
	require.NoError(err)
	stage, paths = ac.ContextDirs()
	require.Equal("", stage)
	require.Len(paths, 0)

	srcs = []string{"src"}
	ac, err = newAddCopyStep(Copy, "", "", "stage", srcs, "", false)
	require.NoError(err)
	stage, paths = ac.ContextDirs()
	require.Equal("stage", stage)
	require.Len(paths, 1)
}
