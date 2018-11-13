package dockerfile

import (
	"strings"
)

// VolumeDirective represents the "VOLUME" dockerfile command.
type VolumeDirective struct {
	*baseDirective
	Volumes []string
}

// Variables:
//   Replaced from ARGs and ENVs from within our stage.
// Formats:
//   VOLUME ["<volume>", ...]
//   VOLUME <volume> ...
func newVolumeDirective(base *baseDirective, state *parsingState) (*VolumeDirective, error) {
	if err := base.replaceVarsCurrStage(state); err != nil {
		return nil, err
	}
	if volumes, ok := parseJSONArray(base.Args); ok {
		return &VolumeDirective{base, volumes}, nil
	}

	return &VolumeDirective{base, strings.Fields(base.Args)}, nil
}

// Add this command to the build stage.
func (d *VolumeDirective) update(state *parsingState) error {
	return state.addToCurrStage(d)
}
