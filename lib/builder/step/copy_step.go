//  Copyright (c) 2018 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package step

import "fmt"

// CopyStep is similar to add, so they depend on a common base.
type CopyStep struct {
	*addCopyStep
}

// NewCopyStep creates a new CopyStep.
func NewCopyStep(
	args, chown, fromStage string, fromPaths []string, toPath string, commit bool,
) (*CopyStep, error) {

	s, err := newAddCopyStep(Copy, args, chown, fromStage, fromPaths, toPath, commit)
	if err != nil {
		return nil, fmt.Errorf("new add/copy step: %s", err)
	}
	return &CopyStep{s}, nil
}
