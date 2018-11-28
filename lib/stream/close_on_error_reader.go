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

package stream

import (
	"fmt"
	"io"
	"syscall"

	"github.com/uber/makisu/lib/log"
)

// CloseOnErrorReader closes reader on any error (includes EOF) and execute a callback function.
// This is useful when caller wants to read until EOF and cleans up after error when Close() is not
// guaranteed to be called.
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
