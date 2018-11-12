package stream

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type readFailer struct {
	closed bool
}

func (readFailer) Read(p []byte) (int, error) {
	return 0, fmt.Errorf("Failed to read bytes")
}

func (reader *readFailer) Close() error {
	reader.closed = true
	return nil
}

func TestCloseOnErrorReader(t *testing.T) {
	require := require.New(t)

	a := 0
	closer := &readFailer{}
	reader := NewCloseOnErrorReader(closer, func() error {
		a = 1
		return nil
	})
	_, err := reader.Read(nil)
	require.Error(err)
	require.Equal(1, a)
	require.True(closer.closed)
}
