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

// CopyDirective represents the "COPY" dockerfile command.
type CopyDirective struct {
	*addCopyDirective
	FromStage string
}

// Variables:
//   Replaced from ARGs and ENVs from within our stage.
// Formats:
//   COPY [--from=<name|index>] [--chown=<user>:<group>] ["<src>",... "<dest>"]
//   COPY [--from=<name|index>] [--chown=<user>:<group>] <src>... <dest>
func newCopyDirective(base *baseDirective, state *parsingState) (Directive, error) {
	if err := base.replaceVarsCurrStage(state); err != nil {
		return nil, err
	}
	args := strings.Fields(base.Args)

	var fromStage string
	if val, ok, err := parseFlag(args[0], "from"); err != nil {
		return nil, base.err(err)
	} else if ok {
		fromStage = val
		args = args[1:]
	} else if len(args) >= 3 {
		if val, ok, err := parseFlag(args[1], "from"); err != nil {
			return nil, base.err(err)
		} else if ok {
			fromStage = val
			args = append([]string{args[0]}, args[2:]...)
		}
	}

	d, err := newAddCopyDirective(base, args)
	if err != nil {
		return nil, err
	}
	return &CopyDirective{d, fromStage}, nil
}

// Add this command to the build stage.
func (d *CopyDirective) update(state *parsingState) error {
	return state.addToCurrStage(d)
}
