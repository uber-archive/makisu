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

// splitArgs splits a whitespace-delimited string into an array of arguments,
// not splitting quoted arguments.
func splitArgs(input string) ([]string, error) {
	var err error
	var state splitArgsState = &splitArgsStateSpace{
		&splitArgsBase{args: make([]string, 0)},
	}
	for i := 0; i < len(input); i++ {
		state, err = state.nextRune(rune(input[i]))
		if err != nil {
			return nil, err
		}
	}
	return state.endOfInput()
}

// splitArgsState defines an interface that a state in the
// key-value parsing state machine must implement.
type splitArgsState interface {
	nextRune(r rune) (splitArgsState, error)
	endOfInput() ([]string, error)
}

// splitArgsBase defines pieces of data that are used & managed by each state.
type splitArgsBase struct {
	args    []string
	currArg string
	escaped bool
}

// splitArgsStateSpace is the starting state for the state machine. It should be entered
// any time an argument has finished being processed.
type splitArgsStateSpace struct{ *splitArgsBase }

// nextRune consumes whitespace until a quote or other non-whitespace character is encountered,
// transitioning to splitArgsStateQuote or splitArgsStateArg, respectively.
func (s *splitArgsStateSpace) nextRune(r rune) (splitArgsState, error) {
	if unicode.IsSpace(r) {
		return s, nil
	} else if r == '"' {
		return &splitArgsStateQuote{s.splitArgsBase}, nil
	} else if r == '\\' {
		s.escaped = true
	} else {
		s.currArg += string(r)
	}
	return &splitArgsStateArg{s.splitArgsBase}, nil
}

// endOfInput returns the args array.
func (s *splitArgsStateSpace) endOfInput() ([]string, error) {
	return s.args, nil
}

// splitArgsStateArg is the state entered on encountering a key value that is not
// wrapped in quotes.
type splitArgsStateArg struct{ *splitArgsBase }

// nextRune appends characters to currArg until a non-escaped whitespace character
// is encountered, consuming currArg and transitioning to splitArgsStateSpace.
func (s *splitArgsStateArg) nextRune(r rune) (splitArgsState, error) {
	if s.escaped {
		if !unicode.IsSpace(r) && r != '"' {
			s.currArg += "\\"
		}
		s.escaped = false
	} else if unicode.IsSpace(r) {
		s.args = append(s.args, s.currArg)
		s.currArg = ""
		return &splitArgsStateSpace{s.splitArgsBase}, nil
	}
	s.currArg += string(r)
	return s, nil
}

// endOfInput appends currArg and returns the args array.
func (s *splitArgsStateArg) endOfInput() ([]string, error) {
	return append(s.args, s.currArg), nil
}

// splitArgsStateQuote is the state entered on encountering a '"' after an '='.
type splitArgsStateQuote struct{ *splitArgsBase }

// nextRune appends to currArg until a non-escaped '"' is encountered,
// transitioning to splitArgsStateEndQuote.
func (s *splitArgsStateQuote) nextRune(r rune) (splitArgsState, error) {
	if s.escaped {
		if r != '"' {
			s.currArg += "\\"
		}
		s.escaped = false
	} else if r == '\\' {
		s.escaped = true
		return s, nil
	} else if r == '"' {
		s.args = append(s.args, s.currArg)
		s.currArg = ""
		return &splitArgsStateEndQuote{s.splitArgsBase}, nil
	}
	s.currArg += string(r)
	return s, nil
}

// endOfInput returns an error, as we cannot terminate in the middle of an argument.
func (s *splitArgsStateQuote) endOfInput() ([]string, error) {
	return nil, fmt.Errorf(
		"unexpected termination: missing '\"' after argument: %s", s.currArg)
}

// splitArgsStateEndQuote is the state entered on encountering a '"' after a value.
type splitArgsStateEndQuote struct{ *splitArgsBase }

// nextRune accepts only a whitespace character, transitioning to splitArgsStateSpace.
func (s *splitArgsStateEndQuote) nextRune(r rune) (splitArgsState, error) {
	if !unicode.IsSpace(r) {
		return nil, errors.New("missing whitespace after argument")
	}
	return &splitArgsStateSpace{s.splitArgsBase}, nil
}

// endOfInput returns the args array.
func (s *splitArgsStateEndQuote) endOfInput() ([]string, error) {
	return s.args, nil
}
