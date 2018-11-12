package tario

import (
	"fmt"
	"io"

	"github.com/klauspost/pgzip"
)

// CompressionLevel is the compression level of image layers.
// Default is pgzip.DefaultCompression.
var CompressionLevel = pgzip.DefaultCompression

var _compressionLevelMap = map[string]int{
	"no":      pgzip.NoCompression,
	"speed":   pgzip.BestSpeed,
	"size":    pgzip.BestCompression,
	"default": pgzip.DefaultCompression,
}

// SetCompressionLevel sets global var CompressionLevel.
func SetCompressionLevel(compressionLevelStr string) error {
	level, ok := _compressionLevelMap[compressionLevelStr]
	if !ok {
		return fmt.Errorf("invalid compression level %s", compressionLevelStr)
	}
	CompressionLevel = level
	return nil
}

// NewGzipWriter returns a new gzip writer with compression level.
func NewGzipWriter(w io.Writer) (io.WriteCloser, error) {
	return pgzip.NewWriterLevel(w, CompressionLevel)
}

// NewGzipReader returns a new gzip reader.
func NewGzipReader(r io.Reader) (io.ReadCloser, error) {
	return pgzip.NewReader(r)
}
