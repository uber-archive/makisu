package dockerfile

// LabelDirective represents the "LABEL" dockerfile command.
type LabelDirective struct {
	*baseDirective
	Labels map[string]string
}

// Variables:
//   Replaced from ARGs and ENVs from within our stage.
// Formats:
//   LABEL <key>=<value> <key>=<value> <key>=<value> ...
func newLabelDirective(base *baseDirective, state *parsingState) (*LabelDirective, error) {
	if err := base.replaceVarsCurrStage(state); err != nil {
		return nil, err
	}
	labels, err := parseKeyVals(base.Args)
	if err != nil {
		return nil, err
	}
	return &LabelDirective{base, labels}, nil
}

// Add this command to the build stage.
func (d *LabelDirective) update(state *parsingState) error {
	return state.addToCurrStage(d)
}
