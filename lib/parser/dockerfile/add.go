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

// AddDirective represents the "ADD" dockerfile command.
type AddDirective struct {
	*addCopyDirective
}

func newAddDirective(base *baseDirective, state *parsingState) (Directive, error) {
	if err := base.replaceVarsCurrStage(state); err != nil {
		return nil, err
	}
	d, err := newAddCopyDirective(base, strings.Fields(base.Args))
	if err != nil {
		return nil, err
	}
	return &AddDirective{d}, nil
}

// Add this command to the build stage.
func (d *AddDirective) update(state *parsingState) error {
	return state.addToCurrStage(d)
}
