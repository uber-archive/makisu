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

type directiveConstructor func(*baseDirective, *parsingState) (Directive, error)

var directiveConstructors = map[string]directiveConstructor{
	"add":         newAddDirective,
	"arg":         newArgDirective,
	"cmd":         newCmdDirective,
	"copy":        newCopyDirective,
	"entrypoint":  newEntrypointDirective,
	"env":         newEnvDirective,
	"expose":      newExposeDirective,
	"from":        newFromDirective,
	"healthcheck": newHealthcheckDirective,
	"label":       newLabelDirective,
	"maintainer":  newMaintainerDirective,
	"run":         newRunDirective,
	"stopsignal":  newStopsignalDirective,
	"user":        newUserDirective,
	"volume":      newVolumeDirective,
	"workdir":     newWorkdirDirective,
}

// newDirective initializes a directive from a line of a Dockerfile and
// the current build state.
// This function also serves to define the variable replacement behavior
// of each directive.
func newDirective(line string, state *parsingState) (Directive, error) {
	base, err := newBaseDirective(line)
	if err != nil {
		return nil, err
	} else if base == nil {
		return nil, nil
	}

	cons, found := directiveConstructors[base.t]
	if !found {
		return nil, base.err(errUnsupportedDirective)
	}
	return cons(base, state)
}
