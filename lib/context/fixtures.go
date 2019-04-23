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

package context

import (
	"io/ioutil"
	"os"

	"github.com/uber/makisu/lib/storage"
	"github.com/uber/makisu/lib/utils/testutil"
)

// BuildContextFixture returns a mock build stage context for testing.
func BuildContextFixture() (*BuildContext, func()) {
	cleanup := &testutil.Cleanup{}
	defer cleanup.Recover()

	rootDir, err := ioutil.TempDir("/tmp", "makisu-test-root")
	if err != nil {
		panic(err)
	}
	cleanup.Add(func() { os.RemoveAll(rootDir) })

	contextDir, err := ioutil.TempDir(rootDir, "context")
	if err != nil {
		panic(err)
	}

	store, c := storage.StoreFixture()
	cleanup.Add(c)

	context, err := NewBuildContext(rootDir, contextDir, store)
	if err != nil {
		panic(err)
	}

	return context, cleanup.Run
}

// BuildContextFixtureWithSampleImage returns a mock build stage context with
// sample image data for testing.
func BuildContextFixtureWithSampleImage() (*BuildContext, func()) {
	cleanup := &testutil.Cleanup{}
	defer cleanup.Recover()

	rootDir, err := ioutil.TempDir("/tmp", "makisu-test-root")
	if err != nil {
		panic(err)
	}
	cleanup.Add(func() { os.RemoveAll(rootDir) })

	contextDir, err := ioutil.TempDir(rootDir, "context")
	if err != nil {
		panic(err)
	}

	sandboxDir, err := ioutil.TempDir("/tmp", "makisu-test-sandbox")
	if err != nil {
		panic(err)
	}
	cleanup.Add(func() { os.RemoveAll(sandboxDir) })

	store, c := storage.StoreFixtureWithSampleImage()
	cleanup.Add(c)

	context, err := NewBuildContext(rootDir, contextDir, store)
	if err != nil {
		panic(err)
	}

	return context, cleanup.Run
}
