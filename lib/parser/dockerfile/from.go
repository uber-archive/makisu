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
	"strings"
)

var errBadAlias = errors.New("Malformed image alias")

// FromDirective represents the "FROM" dockerfile command.
type FromDirective struct {
	*baseDirective
	Image string
	Alias string
}

// Variables:
//   Only replaced using globally defined ARGs (those defined before the first FROM directive.
// Formats:
//   FROM <image> [AS <name>]
func newFromDirective(base *baseDirective, state *parsingState) (Directive, error) {
	if err := base.replaceVarsGlobal(state); err != nil {
		return nil, err
	}
	args := strings.Fields(base.Args)

	var alias string
	if len(args) > 1 {
		if len(args) != 3 || !strings.EqualFold(args[1], "as") {
			return nil, base.err(errBadAlias)
		}
		alias = args[2]
	}

	return &FromDirective{base, args[0], alias}, nil
}

// update:
//   1) Adds a new stage to the parsing state containing the from directive.
//   2) Resets the stage variables.
func (d *FromDirective) update(state *parsingState) error {
	state.addStage(newStage(d))
	state.stageVars = make(map[string]string)
	return nil
}
