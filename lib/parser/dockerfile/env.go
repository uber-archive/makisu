package dockerfile

import (
	"errors"
	"strings"
)

var errMissingSpace = errors.New("Missing space in single value ENV")

// EnvDirective represents the "ENV" dockerfile command.
type EnvDirective struct {
	*baseDirective
	Envs map[string]string
}

// Variables:
//   Replaced from ARGs and ENVs from within our stage.
// Formats:
//   ENV <key> <value>
//   ENV <key>=<value> ...
func newEnvDirective(base *baseDirective, state *parsingState) (*EnvDirective, error) {
	if err := base.replaceVarsCurrStage(state); err != nil {
		return nil, err
	}
	if vars, err := parseKeyVals(base.Args); err == nil {
		return &EnvDirective{base, vars}, nil
	}

	idx := strings.Index(base.Args, " ")
	if idx == -1 || idx == len(base.Args)-1 {
		return nil, base.err(errMissingSpace)
	}

	// Split on the 1st space (including whitespace characters).
	key := base.Args[:idx]
	val := base.Args[idx+1:]

	return &EnvDirective{base, map[string]string{key: val}}, nil
}

// Add this command to the build stage and update stage variables.
func (d *EnvDirective) update(state *parsingState) error {
	for k, v := range d.Envs {
		state.stageVars[k] = v
	}
	return state.addToCurrStage(d)
}
