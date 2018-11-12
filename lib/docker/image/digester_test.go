package image

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/shell"

	"github.com/stretchr/testify/require"
)

func TestDigestFromBytes(t *testing.T) {
	require := require.New(t)

	tmpDir, err := ioutil.TempDir("/tmp", "makisu-digest-test")
	require.NoError(err)
	defer os.RemoveAll(tmpDir)

	targetPath := tmpDir + ".tar"
	err = shell.ExecCommand(log.Infof, log.Errorf, "", "tar", "cvf", targetPath, "--files-from", "/dev/null")
	require.NoError(err)
	defer os.Remove(targetPath)

	f1, err := os.Open(targetPath)
	require.NoError(err)
	defer f1.Close()

	d1, err := NewDigester().FromReader(f1)
	require.NoError(err)

	f2, err := os.Open(targetPath)
	require.NoError(err)
	defer f2.Close()

	b, err := ioutil.ReadAll(f2)
	require.NoError(err)
	d2, err := NewDigester().FromBytes(b)
	require.NoError(err)

	require.Equal(d1, d2)
}
