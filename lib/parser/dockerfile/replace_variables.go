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
)

// replaceVariables replaces all variables in the input string with their values
// as defined in the provided map.
func replaceVariables(input string, vars map[string]string) (string, error) {
	var err error
	var state replaceVarsState = &replaceVarsStateNone{
		&replaceVarsBase{vars: vars},
	}
	for i := 0; i < len(input); i++ {
		state, err = state.nextRune(rune(input[i]))
		if err != nil {
			return "", err
		}
	}
	return state.endOfInput()
}

// replaceVarsState defines an interface that a state in the
// variable replacement state machine must implement.
type replaceVarsState interface {
	nextRune(r rune) (replaceVarsState, error)
	endOfInput() (string, error)
}

// replaceVarsBase defines pieces of data that are used & managed by each state.
type replaceVarsBase struct {
	result         string
	vars           map[string]string
	currVar        string
	varsInProgress []string
	currDefaultCmd rune
	currDefaultVal string
	escaped        bool
}

func (b *replaceVarsBase) reset() {
	b.currVar = ""
	b.currDefaultCmd = 0
	b.currDefaultVal = ""
}

// resolveCurrVar attempts to resolve the current variable using the vars map
// and the default value if it has been set. Returns true if the variable was
// resolved b.vars, else false.
func (b *replaceVarsBase) resolveCurrVar() (string, bool, error) {
	val, ok := b.vars[b.currVar]
	if b.currDefaultCmd == '-' {
		if b.currDefaultVal == "" {
			return "", false, errors.New("missing currDefault variable value")
		}
		if !ok {
			val = b.currDefaultVal
		}
	} else if b.currDefaultCmd == '+' {
		if b.currDefaultVal == "" {
			return "", false, errors.New("missing currDefault variable value")
		}
		if ok {
			val = b.currDefaultVal
		} else {
			val = ""
		}
	} else if b.currDefaultCmd != 0 {
		return "", false, fmt.Errorf("invalid default command: %s", string(b.currDefaultCmd))
	} else if !ok {
		return "", false, nil
	}
	return val, true, nil
}

// replaceVarsStateNone is the starting state for the state machine.
// It should be entered any time a variable has finished being replaced.
type replaceVarsStateNone struct{ *replaceVarsBase }

// nextRune appends all characters that it encounters to the result string until
// encountering a '$', at which point it transitions to replaceVarsStateDollar.
func (s *replaceVarsStateNone) nextRune(r rune) (replaceVarsState, error) {
	if s.escaped {
		if r != '$' {
			s.result += "\\"
		}
		s.escaped = false
	} else if r == '\\' {
		s.escaped = true
		return s, nil
	} else if r == '$' {
		return &replaceVarsStateDollar{s.replaceVarsBase}, nil
	}
	s.result += string(r)
	return s, nil
}

// endOfInput returns the result string.
func (s *replaceVarsStateNone) endOfInput() (string, error) {
	return s.result, nil
}

// replaceVarsStateDollar is the state entered upon encountering a '$'.
type replaceVarsStateDollar struct{ *replaceVarsBase }

// nextRune transitions to replaceVarsStateVarBracket if it encounters a
// '{', else replaceVarsStateVar.
func (s *replaceVarsStateDollar) nextRune(r rune) (replaceVarsState, error) {
	if r == '{' {
		return &replaceVarsStateVarBracket{s.replaceVarsBase}, nil
	}
	s.currVar += string(r)
	return &replaceVarsStateVar{s.replaceVarsBase}, nil
}

// endOfInput returns the result, ending in '$' (equivalent to failing
// to resolve an empty variable).
func (s *replaceVarsStateDollar) endOfInput() (string, error) {
	return s.result + "$", nil
}

// replaceVarsStateVar is the state corresponding to processing a variable of
// the form '$<var>'
type replaceVarsStateVar struct{ *replaceVarsBase }

// resolveCurrVar attempts to resolve the current variable. If it fails, it
// returns the variable itself, including $ and {} where applicable.
func (s *replaceVarsStateVar) resolveCurrVar(recursing bool) (string, error) {
	val, ok, err := s.replaceVarsBase.resolveCurrVar()
	if err != nil {
		return "", err
	}
	if !ok {
		if recursing {
			return "${" + s.currVar + "}", nil
		}
		return "$" + s.currVar, nil
	}
	return val, nil
}

// nextRune appends valid variable characters to currVar. Once an invalid
// character is encountered, we attempt to resolve the current variable and
// transition to replaceVarsStateVarBracket if we are currently processing
// a nested variable, else replaceVarsStateNone.
func (s *replaceVarsStateVar) nextRune(r rune) (replaceVarsState, error) {
	if err := validKeyRune(r); err != nil {
		val, err := s.resolveCurrVar(false)
		if err != nil {
			return nil, err
		}
		// We are not recursing, so just append the result and move on.
		if len(s.varsInProgress) == 0 {
			s.result += val
			if r == '\\' {
				s.escaped = true
			} else if r == '$' {
				s.reset()
				return &replaceVarsStateDollar{s.replaceVarsBase}, nil
			} else {
				s.result += string(r)
			}
			s.reset()
			return &replaceVarsStateNone{s.replaceVarsBase}, nil

			// We are recursing and have encountered a close bracket. If this is the
			// end of the recursion, we need to resolve the calling variable too, as
			// we cannot transition back to the bracket state after having already
			// consumed the close bracket.
		} else if r == '}' {
			s.currVar = s.varsInProgress[len(s.varsInProgress)-1] + val
			s.varsInProgress = s.varsInProgress[0 : len(s.varsInProgress)-1]

			if len(s.varsInProgress) > 0 {
				return &replaceVarsStateVarBracket{s.replaceVarsBase}, nil
			}

			val, err := s.resolveCurrVar(true)
			if err != nil {
				return nil, err
			}
			s.result += val
			s.reset()
			return &replaceVarsStateNone{s.replaceVarsBase}, nil

			// We are recursing and have encountered a colon. This must be the end of
			// the recursion, so if there are still varsInProgress, throw an error.
			// Else, resolve the variable end enter replaceVarsStateVarColon.
		} else if r == ':' {
			if len(s.varsInProgress) != 1 {
				return nil, errors.New("TODO")
			}
			val, err := s.resolveCurrVar(false)
			if err != nil {
				return nil, err
			}
			s.currVar = s.varsInProgress[len(s.varsInProgress)-1] + val
			s.varsInProgress = s.varsInProgress[0 : len(s.varsInProgress)-1]
			return &replaceVarsStateVarColon{s.replaceVarsBase}, nil

		}
		// We are recursing and have encountered another non-variable character.
		// In this case, we return to the bracket state and continue processing.
		s.currVar = s.varsInProgress[len(s.varsInProgress)-1] + val
		s.varsInProgress = s.varsInProgress[0 : len(s.varsInProgress)-1]
		return &replaceVarsStateVarBracket{s.replaceVarsBase}, nil
	}
	s.currVar += string(r)
	return s, nil
}

// endOfInput resolves the current variable and then returns the result.
func (s *replaceVarsStateVar) endOfInput() (string, error) {
	if len(s.varsInProgress) != 0 {
		return "", errors.New("unexpected end of input: in recursive variable resolution")
	}
	val, err := s.resolveCurrVar(false)
	if err != nil {
		return "", err
	}
	return s.result + val, nil
}

// replaceVarsStateVar is the state corresponding to processing a variable of
// the form '${<var>}'
type replaceVarsStateVarBracket struct{ *replaceVarsBase }

// resolveCurrVar attempts to resolve the current variable. If it fails, it returns
// the variable itself, including $ and {} where applicable.
func (s *replaceVarsStateVarBracket) resolveCurrVar() (string, error) {
	val, ok, err := s.replaceVarsBase.resolveCurrVar()
	if err != nil {
		return "", err
	}
	if !ok {
		return "${" + s.currVar + "}", nil
	}
	return val, nil
}

// nextRune appends valid variable characters to currVar until one of these conditions is met:
//   '$' -> replaceVarsStateDollar (recursive resolution)
//   ':' -> replaceVarsStateVarColon
//   '}' -> {replaceVarsStateNone if not recursing, replaceVarsStateBracket if recursing}
func (s *replaceVarsStateVarBracket) nextRune(r rune) (replaceVarsState, error) {
	if r == '$' {
		s.varsInProgress = append(s.varsInProgress, s.currVar)
		s.currVar = ""
		return &replaceVarsStateDollar{s.replaceVarsBase}, nil
	} else if r == ':' {
		if s.currVar == "" {
			return nil, errors.New("missing variable value")
		}
		return &replaceVarsStateVarColon{s.replaceVarsBase}, nil
	} else if r == '}' {
		if len(s.varsInProgress) == 0 {
			val, err := s.resolveCurrVar()
			if err != nil {
				return nil, err
			}
			s.result += val
			s.reset()
			return &replaceVarsStateNone{s.replaceVarsBase}, nil
		}
		val, err := s.resolveCurrVar()
		if err != nil {
			return nil, err
		}
		s.currVar = s.varsInProgress[len(s.varsInProgress)-1] + val
		s.varsInProgress = s.varsInProgress[0 : len(s.varsInProgress)-1]
		return &replaceVarsStateVarBracket{s.replaceVarsBase}, nil
	}
	s.currVar += string(r)
	return s, nil
}

// endOfInput returns an error, as we cannot terminate in the middle of a variable.
func (s *replaceVarsStateVarBracket) endOfInput() (string, error) {
	if s.currVar == "" {
		return "", errors.New("unexpected termination: missing variable value")
	}
	return "", errors.New("missing close bracket after variable")
}

// replaceVarsStateVarColon is the state entered after encountering a ':' in a bracketed
// variable.
type replaceVarsStateVarColon struct{ *replaceVarsBase }

// The first time nextRune is called after a reset, it attempts to set the currDefaultCmd.
// After that, it appends characters to currDefaultVal until a close bracket is encountered.
func (s *replaceVarsStateVarColon) nextRune(r rune) (replaceVarsState, error) {
	if s.currDefaultCmd == 0 {
		if !(r == '-' || r == '+') {
			return nil, fmt.Errorf("invalid default command after ':': %s", string(r))
		}
		s.currDefaultCmd = r
		return s, nil
	} else if s.escaped {
		if r != '}' {
			s.currDefaultVal += "\\"
		}
		s.escaped = false
	} else if r == '\\' {
		s.escaped = true
		return s, nil
	} else if r == '}' {
		val, _, err := s.resolveCurrVar()
		if err != nil {
			return nil, err
		}
		s.result += val
		s.reset()
		return &replaceVarsStateNone{s.replaceVarsBase}, nil
	}
	s.currDefaultVal += string(r)
	return s, nil
}

// endOfInput returns an error, as we cannot terminate in the middle of a variable.
func (s *replaceVarsStateVarColon) endOfInput() (string, error) {
	return "", errors.New("missing close bracket after variable")
}
