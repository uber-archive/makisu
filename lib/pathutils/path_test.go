package pathutils

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAbsPath(t *testing.T) {
	require := require.New(t)

	testPath1 := "./test1/test2/"
	require.Equal("/test1/test2", AbsPath(testPath1))

	testPath2 := "home/test/"
	require.Equal("/home/test", AbsPath(testPath2))

	testPath3 := "/home/test/"
	require.Equal("/home/test", AbsPath(testPath3))
}

func TestRelPath(t *testing.T) {
	require := require.New(t)

	testPath1 := "/test1/test2/"
	require.Equal("test1/test2/", RelPath(testPath1))

	testPath2 := "home/test/"
	require.Equal("home/test/", RelPath(testPath2))
}

func TestSplitPath(t *testing.T) {
	require := require.New(t)

	testPath1 := "/test1/test2/"
	require.Equal([]string{"test1", "test2"}, SplitPath(testPath1))

	testPath2 := "home/test/"
	require.Equal([]string{"home", "test"}, SplitPath(testPath2))

	testPath3 := "/home/test"
	require.Equal([]string{"home", "test"}, SplitPath(testPath3))
}

func TestTrimRoot(t *testing.T) {
	require := require.New(t)

	testRoot1 := "/test/root/"
	testPath1 := "/test/root/test1/test2/"
	trimmed, err := TrimRoot(testPath1, testRoot1)
	require.NoError(err)
	require.Equal("/test1/test2", trimmed)

	testRoot2 := "/test/root/"
	testPath2 := "/test/root/"
	trimmed, err = TrimRoot(testPath2, testRoot2)
	require.NoError(err)
	require.Equal("/", trimmed)

	testRoot3 := "/test/root/"
	testPath3 := "/test/root2/test1"
	trimmed, err = TrimRoot(testPath3, testRoot3)
	require.Error(err)
}

func TestIsDescendantOfAny(t *testing.T) {
	testCases := []struct {
		input  string
		parent string
		result bool
	}{
		{"/a/b", "/a/b", true},
		{"/a/b/", "a/b", true},
		{"a/b", "/a/b/", true},

		{"a/b", "a", true},
		{"a/b", "/a/", true},
		{"a/b/c", "/a/", true},
		{"a/b", "/", true},
		{"a/b", "", true},
		{"/", "/", true},

		{"/x/y/z", "y", false},
		{"/x", "/x/y/z", false},
		{"", "/x", false},
		{"/x_/y", "/x", false},
		{"/x", "/a", false},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			require := require.New(t)
			require.Equal(tc.result, IsDescendantOfAny(tc.input, []string{tc.parent}))
		})
	}
}
