package step

import "fmt"

// AddStep is similar to copy, so they depend on a common base.
type AddStep struct {
	*addCopyStep
}

// NewAddStep creates a new AddStep
func NewAddStep(args, chown string, fromPaths []string, toPath string, commit bool) (*AddStep, error) {
	s, err := newAddCopyStep(Add, args, chown, "", fromPaths, toPath, commit)
	if err != nil {
		return nil, fmt.Errorf("new add/copy step: %s", err)
	}
	return &AddStep{s}, nil
}
