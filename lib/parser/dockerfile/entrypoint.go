package dockerfile

// EntrypointDirective represents the "ENTRYPOINT" dockerfile command.
type EntrypointDirective struct {
	*baseDirective
	Entrypoint []string
}

// Variables:
//   Replaced from ARGs and ENVs from within our stage.
// Formats:
//   ENTRYPOINT ["<executable>", "<param>"...]
//   ENTRYPOINT <command>
func newEntrypointDirective(base *baseDirective, state *parsingState) (*EntrypointDirective, error) {
	if err := base.replaceVarsCurrStage(state); err != nil {
		return nil, err
	}

	if entrypoint, ok := parseJSONArray(base.Args); ok {
		return &EntrypointDirective{base, entrypoint}, nil
	}

	args, err := splitArgs(base.Args)
	if err != nil {
		return nil, base.err(err)
	}
	return &EntrypointDirective{base, args}, nil
}

// Add this command to the build stage.
func (d *EntrypointDirective) update(state *parsingState) error {
	return state.addToCurrStage(d)
}
