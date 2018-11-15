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
