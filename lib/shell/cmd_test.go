package shell

import (
	"bytes"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type bufferedSyncWriter struct {
	b *bytes.Buffer

	sync.Mutex
}

func syncWriterFixture() *bufferedSyncWriter {
	return &bufferedSyncWriter{
		b: &bytes.Buffer{},
	}
}

func (s *bufferedSyncWriter) Write(template string, args ...interface{}) {
	s.Lock()
	defer s.Unlock()

	str := fmt.Sprintf(template, args...)
	s.b.Write([]byte(str))
}

func (s *bufferedSyncWriter) String() string {
	s.Lock()
	defer s.Unlock()

	return s.b.String()
}

func TestExecCommandNoError(t *testing.T) {
	require := require.New(t)
	stdout, stderr := syncWriterFixture(), syncWriterFixture()
	err := ExecCommand(stdout.Write, stderr.Write, ".", "go", "version")
	require.NoError(err)
	require.Empty(stderr.String())
	require.NotEmpty(stdout.String())
}

func TestExecCommandError(t *testing.T) {
	require := require.New(t)
	stdout, stderr := syncWriterFixture(), syncWriterFixture()
	err := ExecCommand(stdout.Write, stderr.Write, ".", "go", "wrong")
	require.Error(err)
	require.NotEmpty(stderr.String())
}
