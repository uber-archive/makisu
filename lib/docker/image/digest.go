package image

import (
	"fmt"
	"io"
	"strings"
)

// Digest is formatted like "<algorithm>:<hex_digest_string>"
// Example:
// 	 sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
type Digest string

// Hex returns the hex part of the digest.
// Example:
//   e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
// This function will panic if the underlying digest doesn't contain ":".
func (d Digest) Hex() string {
	i := strings.Index(string(d), ":")
	return string(d[i+1:])
}

// Equals compares the digest against the layer contained in the reader passed in as input, and
// returns true if the two digests are the same.
func (d Digest) Equals(reader io.ReadCloser) (bool, error) {
	defer reader.Close()
	digester := NewDigester()
	computed, err := digester.FromReader(reader)
	if err != nil {
		return false, fmt.Errorf("digest from reader: %s", err)
	}
	return computed == d, nil
}

// NewEmptyDigest returns a 0 value digest.
func NewEmptyDigest() Digest {
	return Digest("")
}
