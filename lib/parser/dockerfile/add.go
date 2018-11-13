package dockerfile

import (
	"strings"
)

// AddDirective represents the "ADD" dockerfile command.
type AddDirective struct {
	*addCopyDirective
}

func newAddDirective(base *baseDirective, state *parsingState) (*AddDirective, error) {
	if err := base.replaceVarsCurrStage(state); err != nil {
		return nil, err
	}
	d, err := newAddCopyDirective(base, strings.Fields(base.Args))
	if err != nil {
		return nil, err
	}
	return &AddDirective{d}, nil
}

// Add this command to the build stage.
func (d *AddDirective) update(state *parsingState) error {
	return state.addToCurrStage(d)
}
