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
	"fmt"
	"regexp"
	"strings"
)

var (
	commitRegexp     = regexp.MustCompile(`\s*#!\s*commit\s*`)
	whitespaceRegexp = regexp.MustCompile(`\s+`)
)

// baseDirective wraps common info and utilities that all directives depend on.
type baseDirective struct {
	t      string
	Args   string
	Commit bool
}

// newBaseDirective strips and splits the input line. If the line contains only whitespace
// or is empty, returns nil, nil. If the line doesn't contain a directive and arguments,
// returns an error.
func newBaseDirective(line string) (*baseDirective, error) {
	// Handle special commit directive comment.
	// TODO (eoakes): handle escaped comments (\#)
	var commit bool
	if commentIndex := strings.Index(line, "#"); commentIndex != -1 {
		commit = commitRegexp.MatchString(strings.ToLower(line[commentIndex:]))
		line = line[:commentIndex]
	}

	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil, nil
	}
	parts := whitespaceRegexp.Split(trimmed, 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("Failed to parse directive line: '%s'", line)
	}
	t := strings.ToLower(parts[0])
	args := strings.TrimSpace(parts[1])
	return &baseDirective{t, args, commit}, nil
}

// err provides a convenient way to format errors related to parsing
// a directive.
func (d *baseDirective) err(e error) error {
	return &parseError{t: d.t, args: d.Args, msg: e.Error()}
}

// replaceVars replaces the variables in the directive's args string
// using the passed map.
func (d *baseDirective) replaceVars(vars map[string]string) error {
	replaced, err := replaceVariables(d.Args, vars)
	if err != nil {
		return d.err(fmt.Errorf("Failed to replace variables in input: %s", err))
	}
	d.Args = replaced
	return nil
}

// replaceVarsCurrStage replaces variables in the args string using the
// vars map of the current build stage.
func (d *baseDirective) replaceVarsCurrStage(state *parsingState) error {
	if state.stageVars == nil {
		return d.err(errBeforeFirstFrom)
	}
	return d.replaceVars(state.stageVars)
}

// replaceVarsGlobal replaces variables in the args string using the
// global args map.
func (d *baseDirective) replaceVarsGlobal(state *parsingState) error {
	return d.replaceVars(state.globalArgs)
}

// replaceVarsCurrStageOrGlobal replaces variables in the args string as follows:
//   If we have entered a build stage, uses the vars map of the current stage.
//   Else, falls back to the global args map.
func (d *baseDirective) replaceVarsCurrStageOrGlobal(state *parsingState) error {
	vars := state.stageVars
	if vars == nil {
		vars = state.globalArgs
	}
	return d.replaceVars(vars)
}
