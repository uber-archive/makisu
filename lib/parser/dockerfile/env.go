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
	"strings"
)

// EnvDirective represents the "ENV" dockerfile command.
type EnvDirective struct {
	*baseDirective
	Envs map[string]string
}

// Variables:
//   Replaced from ARGs and ENVs from within our stage.
// Formats:
//   ENV <key> <value>
//   ENV <key>=<value> ...
func newEnvDirective(base *baseDirective, state *parsingState) (Directive, error) {
	if err := base.replaceVarsCurrStage(state); err != nil {
		return nil, err
	}
	if vars, err := parseKeyVals(base.Args); err == nil {
		return &EnvDirective{base, vars}, nil
	}

	// Formatted as <key> <value>. Find index of space.
	idx := strings.Index(base.Args, " ")
	if idx == -1 || idx == len(base.Args)-1 {
		return nil, base.err(errMissingSpace)
	}

	// Split on the 1st space (including whitespace characters).
	key := base.Args[:idx]
	val := base.Args[idx+1:]

	return &EnvDirective{base, map[string]string{key: val}}, nil
}

// Add this command to the build stage and update stage variables.
func (d *EnvDirective) update(state *parsingState) error {
	for k, v := range d.Envs {
		state.stageVars[k] = v
	}
	return state.addToCurrStage(d)
}
