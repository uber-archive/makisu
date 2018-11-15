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

func testAndRecover(t *testing.T, f func(), name string) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("%s: should have panicked", name)
		}
	}()
	f()
}

func TestFixtures(t *testing.T) {
	require := require.New(t)

	// Valid.
	require.NotNil(FromStepFixture("", "image", "alias"))
	require.NotNil(CopyStepFixture("", "", []string{"."}, "/", false))
	require.NotNil(CopyStepFixtureNoChown("", "", []string{"."}, "/", false))
	require.NotNil(AddStepFixture("", []string{"."}, "/", false))
	require.NotNil(AddStepFixtureNoChown("", []string{"."}, "/", false))

	// Invalid.
	testAndRecover(t, func() { FromStepFixture("", "image:", "") }, "FROM bad image")
	testAndRecover(t, func() { CopyStepFixture("", "", []string{".", "."}, "/file", false) }, "invalid COPY")
	testAndRecover(t, func() { CopyStepFixtureNoChown("", "", []string{".", "."}, "/file", false) }, "invalid COPY")
	testAndRecover(t, func() { AddStepFixture("", []string{".", "."}, "/file", false) }, "invalid COPY")
	testAndRecover(t, func() { AddStepFixtureNoChown("", []string{".", "."}, "/file", false) }, "invalid COPY")
}
