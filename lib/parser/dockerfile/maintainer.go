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

// MaintainerDirective represents the "MAINTAINER" dockerfile command.
type MaintainerDirective struct {
	*baseDirective
	Author string
}

// Formats:
//   MAINTAINER <value> ...
func newMaintainerDirective(base *baseDirective, state *parsingState) (Directive, error) {
	return &MaintainerDirective{base, base.Args}, nil
}

// Add this command to the build stage.
func (d *MaintainerDirective) update(state *parsingState) error {
	return state.addToCurrStage(d)
}
