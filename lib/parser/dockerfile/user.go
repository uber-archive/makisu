package dockerfile

import (
	"strings"
)

// UserDirective represents the "USER" dockerfile command.
type UserDirective struct {
	*baseDirective
	User string
}

// Variables:
//   Replaced from ARGs and ENVs from within our stage.
// Formats:
//   USER <user>[:<group>]
func newUserDirective(base *baseDirective, state *parsingState) (*UserDirective, error) {
	if err := base.replaceVarsCurrStage(state); err != nil {
		return nil, err
	}
	args := strings.Fields(base.Args)
	if len(args) != 1 {
		return nil, base.err(errNotExactlyOneArg)
	}

	return &UserDirective{base, args[0]}, nil
}

// Add this command to the build stage.
func (d *UserDirective) update(state *parsingState) error {
	return state.addToCurrStage(d)
}
