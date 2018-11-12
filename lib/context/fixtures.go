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
