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
	"strings"
	"sync"
)

// ConcurrentMultiWriter is a concurrent implementation of the io.MultiWriter of the standard library.
type ConcurrentMultiWriter struct {
	writers []io.Writer
}

// NewConcurrentMultiWriter returns a new ConcurrentMultiWriter.
func NewConcurrentMultiWriter(writers ...io.Writer) *ConcurrentMultiWriter {
	return &ConcurrentMultiWriter{writers: writers}
}

// Write implements io.Writer.
func (w *ConcurrentMultiWriter) Write(p []byte) (int, error) {
	wg := sync.WaitGroup{}
	n := len(p)

	var mu sync.Mutex
	var errMsgs []string
	for _, writer := range w.writers {
		wg.Add(1)
		go func(writer io.Writer) {
			defer wg.Done()
			numBytes, thisError := writer.Write(p)
			if thisError != nil {
				mu.Lock()
				defer mu.Unlock()
				errMsgs = append(errMsgs, thisError.Error())
				return
			}
			if numBytes != len(p) {
				mu.Lock()
				defer mu.Unlock()
				errMsgs = append(errMsgs, thisError.Error())
				return
			}
		}(writer)
	}
	wg.Wait()

	if errMsgs != nil {
		return -1, fmt.Errorf("failed to write: %s", strings.Join(errMsgs, ", "))
	}
	return n, nil
}
