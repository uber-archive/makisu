package dockerfile

import (
	"strings"
)

// WorkdirDirective represents the "WORKDIR" dockerfile command.
type WorkdirDirective struct {
	*baseDirective
	WorkingDir string
}

// Variables:
//   Replaced from ARGs and ENVs from within our stage.
// Formats:
//   WORKDIR <path>
func newWorkdirDirective(base *baseDirective, state *parsingState) (*WorkdirDirective, error) {
	if err := base.replaceVarsCurrStage(state); err != nil {
		return nil, err
	}
	args := strings.Fields(base.Args)
	if len(args) != 1 {
		return nil, base.err(errNotExactlyOneArg)
	}

	return &WorkdirDirective{base, args[0]}, nil
}

// Add this command to the build stage.
func (d *WorkdirDirective) update(state *parsingState) error {
	return state.addToCurrStage(d)
}
