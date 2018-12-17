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

package dockerfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

var (
	errBeforeFirstFrom      = errors.New("Invalid directive before first build stage (FROM)")
	errMalformedChown       = errors.New("Malformed chown argument")
	errMalformedKeyVal      = errors.New("Malformed key/value pairs")
	errMissingArgs          = errors.New("Missing arguments")
	errMissingSpace         = errors.New("Missing space in single variable ENV")
	errNotExactlyOneArg     = errors.New("Expected exactly one argument")
	errUnsupportedDirective = errors.New("Unsupported directive type")
)

func parseFlag(s string, name string) (string, bool, error) {
	flag := "--" + name + "="
	if !strings.HasPrefix(s, flag) {
		return "", false, nil
	}
	if len(s) == len(flag) {
		return "", false, fmt.Errorf("Missing value for flag: %s", name)
	}
	return s[len(flag):], true, nil
}

func parseJSONArray(s string) (l []string, ok bool) {
	err := json.NewDecoder(strings.NewReader(s)).Decode(&l)
	return l, err == nil
}

// validKeyRune returns an error if the rune is not a valid key character.
func validKeyRune(r rune) error {
	if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '-' || r == '_' || r == '.' {
		return nil
	}
	return fmt.Errorf("invalid character in variable key: %s", string(r))
}

// parseError represents an error that occurred while trying to parse a directive string.
type parseError struct {
	t    string
	args string
	msg  string
}

// Error returns a formatted error string.
func (e *parseError) Error() string {
	return fmt.Sprintf("failed to parse '%s' directive with args '%s': %s",
		strings.ToUpper(e.t), e.args, e.msg)
}

// String returns a formatted error string.
func (e *parseError) String() string { return e.Error() }
