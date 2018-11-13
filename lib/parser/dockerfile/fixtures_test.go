package dockerfile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFromDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(FromDirectiveFixture("image as alias", "image", "alias"))
}

func TestRunDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(RunDirectiveFixture("ls /", "ls /"))
}

func TestCmdDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(CmdDirectiveFixture("ls /", []string{"ls", "/"}))
}

func TestLabelDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(LabelDirectiveFixture("label key=val", map[string]string{"key": "val"}))
}

func TestExposeDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(ExposeDirectiveFixture("expose 80/tcp", []string{"80/tcp"}))
}

func TestCopyDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(CopyDirectiveFixture("src1 src2 dst/", "", "", []string{"src1", "src2"}, "dst/"))
}

func TestEntrypointDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(EntrypointDirectiveFixture("ls /", []string{"ls", "/"}))
}

func TestEnvDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(EnvDirectiveFixture("key=val", map[string]string{"key": "val"}))
}

func TestUserDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(UserDirectiveFixture("user", "user"))
}

func TestVolumeDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(VolumeDirectiveFixture("volume /tmp:/tmp", []string{"/tmp:/tmp"}))
}

func TestWorkdirDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(WorkdirDirectiveFixture("/home", "/home"))
}

func TestAddDirectiveFixture(t *testing.T) {
	require := require.New(t)
	require.NotNil(AddDirectiveFixture("src1 src2 dst/", "", []string{"src1", "src2"}, "dst/"))
}
