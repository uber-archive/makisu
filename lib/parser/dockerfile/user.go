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

// UserDirective represents the "USER" dockerfile command.
type UserDirective struct {
	*baseDirective
	User string
}

// Variables:
//   Replaced from ARGs and ENVs from within our stage.
// Formats:
//   USER <user>[:<group>]
func newUserDirective(base *baseDirective, state *parsingState) (Directive, error) {
	if err := base.replaceVarsCurrStage(state); err != nil {
		return nil, err
	}
	args := strings.Fields(base.Args)
	if len(args) != 1 {
		return nil, base.err(errNotExactlyOneArg)
	}

	return &UserDirective{base, args[0]}, nil
}

// Add this command to the build stage.
func (d *UserDirective) update(state *parsingState) error {
	return state.addToCurrStage(d)
}
