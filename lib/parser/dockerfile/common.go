package dockerfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

var (
	errUnsupportedDirective = errors.New("Unsupported directive type")
	errBeforeFirstFrom      = errors.New("Invalid directive before first build stage (FROM)")
	errMissingArgs          = errors.New("Missing arguments")
	errMalformedChown       = errors.New("Malformed chown argument")
	errMalformedKeyVal      = errors.New("Malformed key/value pairs")
	errNotExactlyOneArg     = errors.New("Expected exactly one argument")
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
	return fmt.Sprintf("failed to parse %s directive with args '%s': %s",
		strings.ToUpper(e.t), e.args, e.msg)
}
