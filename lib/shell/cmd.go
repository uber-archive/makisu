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
			outStream("Failed to stream stdout from command: %v\n", err)
		}
	}()

	go func() {
		if err := readerToStream(errReader, errStream); err != nil {
			errStream("Failed to stream stderr from command: %v\n", err)
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
