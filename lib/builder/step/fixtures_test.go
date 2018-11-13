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
