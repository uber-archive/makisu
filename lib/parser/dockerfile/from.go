package dockerfile

import (
	"errors"
	"strings"
)

var errBadAlias = errors.New("Malformed image alias")

// FromDirective represents the "FROM" dockerfile command.
type FromDirective struct {
	*baseDirective
	Image string
	Alias string
}

// Variables:
//   Only replaced using globally defined ARGs (those defined before the first FROM directive.
// Formats:
//   FROM <image> [AS <name>]
func newFromDirective(base *baseDirective, state *parsingState) (*FromDirective, error) {
	if err := base.replaceVarsGlobal(state); err != nil {
		return nil, err
	}
	args := strings.Fields(base.Args)

	var alias string
	if len(args) > 1 {
		if len(args) != 3 || !strings.EqualFold(args[1], "as") {
			return nil, base.err(errBadAlias)
		}
		alias = args[2]
	}

	return &FromDirective{base, args[0], alias}, nil
}

// update:
//   1) Adds a new stage to the parsing state containing the from directive.
//   2) Resets the stage variables.
func (d *FromDirective) update(state *parsingState) error {
	state.addStage(newStage(d))
	state.stageVars = make(map[string]string)
	return nil
}
