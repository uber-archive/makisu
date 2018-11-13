package dockerfile

// CmdDirective represents the "CMD" dockerfile command.
type CmdDirective struct {
	*baseDirective
	Cmd []string
}

// Variables:
//   Replaced from ARGs and ENVs from within our stage.
// Formats:
//   CMD ["<executable>", "<param>"...]
//   CMD ["<param>"...]
//   CMD <command> <param>...
func newCmdDirective(base *baseDirective, state *parsingState) (*CmdDirective, error) {
	if err := base.replaceVarsCurrStage(state); err != nil {
		return nil, err
	}
	if cmd, ok := parseJSONArray(base.Args); ok {
		return &CmdDirective{base, cmd}, nil
	}

	args, err := splitArgs(base.Args)
	if err != nil {
		return nil, base.err(err)
	}
	return &CmdDirective{base, args}, nil
}

// Add this command to the build stage.
func (d *CmdDirective) update(state *parsingState) error {
	return state.addToCurrStage(d)
}
