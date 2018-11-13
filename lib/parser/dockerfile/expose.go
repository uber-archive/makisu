package dockerfile

import (
	"strings"
)

// ExposeDirective represents the "EXPOSE" dockerfile command.
type ExposeDirective struct {
	*baseDirective
	Ports []string
}

// Variables:
//   Replaced from ARGs and ENVs from within our stage.
// Formats:
//   EXPOSE <port>[/<protocol]...
func newExposeDirective(base *baseDirective, state *parsingState) (*ExposeDirective, error) {
	if err := base.replaceVarsCurrStage(state); err != nil {
		return nil, err
	}
	return &ExposeDirective{base, strings.Fields(base.Args)}, nil
}

// Add this command to the build stage.
func (d *ExposeDirective) update(state *parsingState) error {
	return state.addToCurrStage(d)
}
