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
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

type writeFailer struct{}

func (writeFailer) Write(p []byte) (int, error) {
	return 0, fmt.Errorf("Failed to write bytes")
}

func TestMultiWriter(t *testing.T) {
	require := require.New(t)

	buffers := make([]*bytes.Buffer, 10)
	ptrs := make([]io.Writer, 10)
	for i := range buffers {
		buffers[i] = &bytes.Buffer{}
		ptrs[i] = buffers[i]
	}
	multi := NewConcurrentMultiWriter(ptrs...)
	testString := "THIS IS MY TEST STRING"
	multi.Write([]byte(testString))
	for _, buffer := range buffers {
		require.Equal(testString, buffer.String())
	}
}

func TestMultiWriterFailure(t *testing.T) {
	require := require.New(t)

	writers := []io.Writer{writeFailer{}, &bytes.Buffer{}}
	multi := NewConcurrentMultiWriter(writers...)
	_, err := multi.Write([]byte("TEST"))
	require.Error(err)
}
