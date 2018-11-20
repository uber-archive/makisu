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

// DigestPairs is a list of DigestPair
type DigestPairs []*DigestPair

// DigestPairMap is a map from string to DigestPairs
type DigestPairMap map[string]DigestPairs

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
