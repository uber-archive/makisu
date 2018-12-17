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
	"fmt"
	"io"
	"os/exec"
	"syscall"
)

// ShellStreamBufferSize is the size of the output buffers when streaming command stdout and stderr
const ShellStreamBufferSize = 1 << 20

// ExecCommand exec a command given workingDir, cmd and args, returns error if cmd fails
func ExecCommand(outStream, errStream func(string, ...interface{}), workingDir, cmdName string, cmdArgs ...string) error {
	cmd := exec.Command(cmdName, cmdArgs...)
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	outReader, outWriter := io.Pipe()
	errReader, errWriter := io.Pipe()
	cmd.Stdout, cmd.Stderr = outWriter, errWriter

	go func() {
		if err := readerToStream(outReader, outStream); err != nil {
			outStream("Failed to stream stdout from command: %s\n", err)
		}
	}()

	go func() {
		if err := readerToStream(errReader, errStream); err != nil {
			errStream("Failed to stream stderr from command: %s\n", err)
		}
	}()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cmd start: %s", err)
	} else if err := cmd.Wait(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// Command exited with code other than 0.
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode := ws.ExitStatus()
			errStream("Command exited with %d\n", exitCode)
			return exitError
		}
		return fmt.Errorf("cmd wait: %s", err)
	}
	return nil
}

func readerToStream(reader io.Reader, stream func(string, ...interface{})) error {
	buffer := make([]byte, ShellStreamBufferSize)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			stream("%s", buffer[:n])
		}

		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
	}
}
