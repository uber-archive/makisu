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
