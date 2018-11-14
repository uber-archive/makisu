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

// Directive defines a directive parsed from a line from a Dockerfile.
type Directive interface {
	update(*parsingState) error
}

// newDirective initializes a directive from a line of a Dockerfile and
// the current build state.
// This function also serves to define the variable replacement behavior
// of each directive.
func newDirective(line string, state *parsingState) (Directive, error) {
	base, err := newBaseDirective(line)
	if err != nil {
		return nil, err
	}
	if base == nil {
		return nil, nil
	}

	switch base.t {
	case "add":
		return newAddDirective(base, state)
	case "arg":
		return newArgDirective(base, state)
	case "cmd":
		return newCmdDirective(base, state)
	case "copy":
		return newCopyDirective(base, state)
	case "entrypoint":
		return newEntrypointDirective(base, state)
	case "env":
		return newEnvDirective(base, state)
	case "expose":
		return newExposeDirective(base, state)
	case "from":
		return newFromDirective(base, state)
	case "label":
		return newLabelDirective(base, state)
	case "maintainer":
		return newMaintainerDirective(base, state)
	case "run":
		return newRunDirective(base, state)
	case "user":
		return newUserDirective(base, state)
	case "volume":
		return newVolumeDirective(base, state)
	case "workdir":
		return newWorkdirDirective(base, state)
	default:
		return nil, base.err(errUnsupportedDirective)
	}
}
