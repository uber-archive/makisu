package stream

import (
	"fmt"
	"io"
	"syscall"

	"github.com/uber/makisu/lib/log"
)

// CloseOnErrorReader closes reader on any error (includes EOF) and execute a callback function.
// This is useful when caller wants to read until EOF and cleans up after error when Close() is not
// guranteed to be called.
type CloseOnErrorReader struct {
	readCloser io.ReadCloser
	callBack   func() error
}

// NewCloseOnErrorReader creates a new CloseOnErrorReader
func NewCloseOnErrorReader(readCloser io.ReadCloser, callBack func() error) io.Reader {
	return &CloseOnErrorReader{
		readCloser: readCloser,
		callBack:   callBack,
	}
}

// Read implements io.Reader.Read
func (r *CloseOnErrorReader) Read(p []byte) (int, error) {
	n, err := r.readCloser.Read(p)
	if err != nil {
		r.Close()
		if err == io.EOF {
			return n, io.EOF
		}
		return 0, fmt.Errorf("read: %s", err)
	}
	return n, nil
}

// Close implement io.ReadCloser.Close
func (r *CloseOnErrorReader) Close() error {
	defer func() {
		if r.callBack == nil {
			return
		}
		if err := r.callBack(); err != nil {
			log.Error(err)
		}
	}()

	if err := r.readCloser.Close(); err != nil && err != syscall.EINVAL {
		return fmt.Errorf("close: %s", err)
	}
	return nil
}
