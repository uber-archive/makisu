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

type argDirective struct {
	*baseDirective
	name       string
	defaultVal string
}

// Variables:
//   If we have not yet entered the first stage (encountered a FROM directive), variables are
//   replaced using other globally-defined ARGs.
//   Else, variables are replaced from ARGs and ENVs from within our stage.
// Formats:
//   ARG <name>[=<default value>]
func newArgDirective(base *baseDirective, state *parsingState) (*argDirective, error) {
	if err := base.replaceVarsCurrStageOrGlobal(state); err != nil {
		return nil, err
	}
	if vars, err := parseKeyVals(base.Args); err == nil {
		if len(vars) != 1 {
			return nil, base.err(errNotExactlyOneArg)
		}
		var name string
		var defaultVal string
		for k, v := range vars {
			name = k
			defaultVal = v
		}
		return &argDirective{base, name, defaultVal}, nil
	}

	args, err := splitArgs(base.Args)
	if err != nil {
		return nil, base.err(err)
	}
	if len(args) != 1 {
		return nil, base.err(errNotExactlyOneArg)
	}

	return &argDirective{base, base.Args, ""}, nil
}

// ARGs only update global/stage variables.
// If we have not yet entered the first stage (encountered a FROM directive), we update
// the global args map.
// Else, we update the current stage variables.
// In either case, we only update the variable if it has a default value or a value is passed.
func (d *argDirective) update(state *parsingState) error {
	vars := state.stageVars
	if vars == nil {
		vars = state.globalArgs
	}
	if val, ok := state.passedArgs[d.name]; ok {
		vars[d.name] = val
	} else if d.defaultVal != "" {
		vars[d.name] = d.defaultVal
	}
	return nil
}
