package dockerfile

// MaintainerDirective represents the "MAINTAINER" dockerfile command.
type MaintainerDirective struct {
	*baseDirective
	author string
}

// Formats:
//   MAINTAINER <value> ...
func newMaintainerDirective(base *baseDirective, state *parsingState) (*MaintainerDirective, error) {
	return &MaintainerDirective{base, base.Args}, nil
}

// Add this command to the build stage.
func (d *MaintainerDirective) update(state *parsingState) error {
	return state.addToCurrStage(d)
}
