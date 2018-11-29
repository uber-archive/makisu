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
	"add": func(base *baseDirective, state *parsingState) (Directive, error) { return newAddDirective(base, state) },
	"arg": func(base *baseDirective, state *parsingState) (Directive, error) { return newArgDirective(base, state) },
	"cmd": func(base *baseDirective, state *parsingState) (Directive, error) { return newCmdDirective(base, state) },
	"copy": func(base *baseDirective, state *parsingState) (Directive, error) {
		return newCopyDirective(base, state)
	},
	"entrypoint": func(base *baseDirective, state *parsingState) (Directive, error) {
		return newEntrypointDirective(base, state)
	},
	"env": func(base *baseDirective, state *parsingState) (Directive, error) { return newEnvDirective(base, state) },
	"expose": func(base *baseDirective, state *parsingState) (Directive, error) {
		return newExposeDirective(base, state)
	},
	"from": func(base *baseDirective, state *parsingState) (Directive, error) {
		return newFromDirective(base, state)
	},
	"label": func(base *baseDirective, state *parsingState) (Directive, error) {
		return newLabelDirective(base, state)
	},
	"maintainer": func(base *baseDirective, state *parsingState) (Directive, error) {
		return newMaintainerDirective(base, state)
	},
	"run": func(base *baseDirective, state *parsingState) (Directive, error) { return newRunDirective(base, state) },
	"stopsignal": func(base *baseDirective, state *parsingState) (Directive, error) {
		return newStopsignalDirective(base, state)
	},
	"user": func(base *baseDirective, state *parsingState) (Directive, error) {
		return newUserDirective(base, state)
	},
	"volume": func(base *baseDirective, state *parsingState) (Directive, error) {
		return newVolumeDirective(base, state)
	},
	"workdir": func(base *baseDirective, state *parsingState) (Directive, error) {
		return newWorkdirDirective(base, state)
	},
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
