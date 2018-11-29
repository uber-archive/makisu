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

// RunDirective represents the "RUN" dockerfile command.
type RunDirective struct {
	*baseDirective
	Cmd string
}

// Variables:
//   Replaced from ARGs and ENVs from within our stage.
// Formats:
//   RUN ["<executable>", "<param>"...]
//   RUN ["<param>"...]
//   RUN <command>
func newRunDirective(base *baseDirective, state *parsingState) (Directive, error) {
	if err := base.replaceVarsCurrStage(state); err != nil {
		return nil, err
	}
	if cmd, ok := parseJSONArray(base.Args); ok {
		return &RunDirective{base, strings.Join(cmd, " ")}, nil
	}

	return &RunDirective{base, base.Args}, nil
}

// Add this command to the build stage.
func (d *RunDirective) update(state *parsingState) error {
	return state.addToCurrStage(d)
}
