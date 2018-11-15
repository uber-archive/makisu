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
	"errors"
	"fmt"
	"unicode"
)

var (
	errMissingSeparator = errors.New("missing separator")
	errMissingValue     = errors.New("missing value")
)

// parseKeyVals parses a whitespace-delimited string consisting of <key>=<value>
// pairs into a map. Both keys and values may optionally contain whitespace by
// escaping them using '\' or using double quotes.
func parseKeyVals(input string) (map[string]string, error) {
	var err error
	var state parseKVsState = &parseKVsStateSpace{
		&parseKVsBase{vars: make(map[string]string)},
	}
	for i := 0; i < len(input); i++ {
		state, err = state.nextRune(rune(input[i]))
		if err != nil {
			return nil, err
		}
	}
	return state.endOfInput()
}

// parseKVsState defines an interface that a state in the
// key-value parsing state machine must implement.
type parseKVsState interface {
	nextRune(r rune) (parseKVsState, error)
	endOfInput() (map[string]string, error)
}

// parseKVsBase defines pieces of data that are used & managed by each state.
type parseKVsBase struct {
	vars    map[string]string
	currKey string
	currVal string
	escaped bool
}

// consumeCurrKV sets currKey=currVal in the vars map and resets them.
func (s *parseKVsBase) consumeCurrKV() error {
	if s.currVal == "" {
		return errMissingValue
	}
	s.vars[s.currKey] = s.currVal
	s.currKey = ""
	s.currVal = ""
	return nil
}

// parseKVsStateSpace is the starting state for the state machine. It should be entered
// any time a key-value pair has finished being processed.
type parseKVsStateSpace struct{ *parseKVsBase }

// nextRune consumes whitespace until a quote or valid key character is encountered,
// transitioning to parseKVsStateKeyQuote or parseKVsStateKey, respectively.
func (s *parseKVsStateSpace) nextRune(r rune) (parseKVsState, error) {
	if unicode.IsSpace(r) {
		return s, nil
	} else if err := validKeyRune(r); err != nil {
		return nil, err
	}
	s.currKey += string(r)
	return &parseKVsStateKey{s.parseKVsBase}, nil
}

// endOfInput returns the KV map.
func (s *parseKVsStateSpace) endOfInput() (map[string]string, error) {
	return s.vars, nil
}

// parseKVsStateKey is the state entered on encountering a key value that is not
// wrapped in quotes.
type parseKVsStateKey struct{ *parseKVsBase }

// nextRune appends valid key characters to currKey until '=' is encountered,
// transitioning to parseKVsStateEquals.
func (s *parseKVsStateKey) nextRune(r rune) (parseKVsState, error) {
	if r == '=' {
		return &parseKVsStateEquals{s.parseKVsBase}, nil
	} else if err := validKeyRune(r); err != nil {
		return nil, err
	}
	s.currKey += string(r)
	return s, nil
}

// endOfInput returns an error, as we cannot terminate in the middle of a key.
func (s *parseKVsStateKey) endOfInput() (map[string]string, error) {
	return nil, fmt.Errorf("unexpected termination: expected '=<value>' after key: %s", s.currKey)
}

// parseKVsStateEquals is the state entered on encountering an '=' after a key.
type parseKVsStateEquals struct{ *parseKVsBase }

// nextRune accepts either a single '"' or valid character, transitioning
// to parseKVsStateValQuote or parseKVsStateVal, respectively.
func (s *parseKVsStateEquals) nextRune(r rune) (parseKVsState, error) {
	if r == '"' {
		return &parseKVsStateValQuote{s.parseKVsBase}, nil
	} else if r == '\\' {
		s.escaped = true
		return &parseKVsStateVal{s.parseKVsBase}, nil
	}
	s.currVal += string(r)
	return &parseKVsStateVal{s.parseKVsBase}, nil
}

// parseKVsStateVal appends value characters until a whitespace character is encountered,
// consuming the current KV pair and transitioning to parseKVsStateSpace.
type parseKVsStateVal struct{ *parseKVsBase }

func (s *parseKVsStateVal) nextRune(r rune) (parseKVsState, error) {
	if s.escaped {
		if !unicode.IsSpace(r) && r != '"' {
			s.currVal += string('\\')
		}
		s.escaped = false
	} else if r == '\\' {
		s.escaped = true
		return s, nil
	} else if unicode.IsSpace(r) {
		if err := s.consumeCurrKV(); err != nil {
			return nil, err
		}
		return &parseKVsStateSpace{s.parseKVsBase}, nil
	}
	s.currVal += string(r)
	return s, nil
}

// endOfInput consumes the existing KV pair and then returns the KV map.
func (s *parseKVsStateVal) endOfInput() (map[string]string, error) {
	if err := s.consumeCurrKV(); err != nil {
		return nil, err
	}
	return s.vars, nil
}

// endOfInput returns an error, as we cannot terminate without a value.
func (s *parseKVsStateEquals) endOfInput() (map[string]string, error) {
	return nil, fmt.Errorf("unexpected termination: expected '=<value>' after key: %s", s.currKey)
}

// parseKVsStateValQuote is the state entered on encountering a '"' after an '='.
type parseKVsStateValQuote struct{ *parseKVsBase }

// consumeCurrKV sets currKey="currVal" in the vars map and resets them.
// It allows current value to be empty.
func (s *parseKVsStateValQuote) consumeCurrKV() error {
	s.vars[s.currKey] = s.currVal
	s.currKey = ""
	s.currVal = ""
	return nil
}

// nextRune appends valid value characters to currVal until '"' is encountered,
// consuming the current KV pair and transitioning to parseKVsStateValEndQuote.
func (s *parseKVsStateValQuote) nextRune(r rune) (parseKVsState, error) {
	if s.escaped {
		if r != '"' {
			s.currVal += "\\"
		}
		s.escaped = false
	} else if r == '\\' {
		s.escaped = true
		return &parseKVsStateValQuote{s.parseKVsBase}, nil
	} else if r == '"' {
		if err := s.consumeCurrKV(); err != nil {
			return nil, err
		}
		return &parseKVsStateValEndQuote{s.parseKVsBase}, nil
	}
	s.currVal += string(r)
	return s, nil
}

// endOfInput returns an error, as we cannot terminate in the middle of a value.
func (s *parseKVsStateValQuote) endOfInput() (map[string]string, error) {
	return nil, fmt.Errorf(
		`unexpected termination: missing '"' after value: '%s' for key: '%s'`, s.currVal, s.currKey)
}

// parseKVsStateValEndQuote is the state entered on encountering a '"' after a value.
type parseKVsStateValEndQuote struct{ *parseKVsBase }

// nextRune accepts only a whitespace character, transitioning to parseKVsStateSpace.
func (s *parseKVsStateValEndQuote) nextRune(r rune) (parseKVsState, error) {
	if !unicode.IsSpace(r) {
		return nil, errors.New("missing whitespace after value")
	}
	return &parseKVsStateSpace{s.parseKVsBase}, nil
}

// endOfInput returns the KV map.
func (s *parseKVsStateValEndQuote) endOfInput() (map[string]string, error) {
	return s.vars, nil
}
