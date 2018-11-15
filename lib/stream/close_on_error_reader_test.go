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
