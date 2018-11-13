package dockerfile

import (
	"strings"
)

// RunDirective represents the "RUN" dockerfile command.
type RunDirective struct {
	*baseDirective
	Cmd string
}

// Variables:
//   Replaced from ARGs and ENVs from within our stage.
// Formats:
//   RUN ["<executable>", "<param>"...]
//   RUN ["<param>"...]
//   RUN <command>
func newRunDirective(base *baseDirective, state *parsingState) (*RunDirective, error) {
	if err := base.replaceVarsCurrStage(state); err != nil {
		return nil, err
	}
	if cmd, ok := parseJSONArray(base.Args); ok {
		return &RunDirective{base, strings.Join(cmd, " ")}, nil
	}

	return &RunDirective{base, base.Args}, nil
}

// Add this command to the build stage.
func (d *RunDirective) update(state *parsingState) error {
	return state.addToCurrStage(d)
}
