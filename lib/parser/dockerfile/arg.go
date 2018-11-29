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

// ArgDirective represents the "ARG" dockerfile command.
type ArgDirective struct {
	*baseDirective
	Name        string
	DefaultVal  string
	ResolvedVal *string
}

// Variables:
//   If we have not yet entered the first stage (encountered a FROM directive),
//   variables are replaced using other globally-defined ARGs.
//   Else, variables are replaced from ARGs and ENVs from within our stage.
// Formats:
//   ARG <name>[=<default value>]
func newArgDirective(base *baseDirective, state *parsingState) (Directive, error) {
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
		return &ArgDirective{base, name, defaultVal, nil}, nil
	}

	args, err := splitArgs(base.Args)
	if err != nil {
		return nil, base.err(err)
	}
	if len(args) != 1 {
		return nil, base.err(errNotExactlyOneArg)
	}

	return &ArgDirective{base, base.Args, "", nil}, nil
}

// ARGs only update global/stage variables.
// If we have not yet entered the first stage (encountered a FROM directive), we update
// the global args map.
// Else, we update the current stage variables.
// In either case, we only update the variable if it has a default value or a value is passed.
func (d *ArgDirective) update(state *parsingState) error {
	var global bool
	vars := state.stageVars
	if vars == nil {
		global = true
		vars = state.globalArgs
	}
	if val, ok := state.passedArgs[d.Name]; ok {
		vars[d.Name] = val
		d.ResolvedVal = &val
	} else if d.DefaultVal != "" {
		vars[d.Name] = d.DefaultVal
		d.ResolvedVal = &d.DefaultVal
	}
	if !global {
		return state.addToCurrStage(d)
	}
	return nil
}
