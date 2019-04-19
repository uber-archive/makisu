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
	err := ExecCommand(stdout.Write, stderr.Write, ".", "", "go", "version")
	require.NoError(err)
	require.Empty(stderr.String())
	require.NotEmpty(stdout.String())
}

func TestExecCommandError(t *testing.T) {
	require := require.New(t)
	stdout, stderr := syncWriterFixture(), syncWriterFixture()
	err := ExecCommand(stdout.Write, stderr.Write, ".", "", "go", "wrong")
	require.Error(err)
	require.NotEmpty(stderr.String())
}
