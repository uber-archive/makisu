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

import (
	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
)

// ArgStep implements BuildStep and execute ARG directive
type ArgStep struct {
	*baseStep

	name        string
	resolvedVal *string
}

// NewArgStep returns a BuildStep from given arguments.
func NewArgStep(args string, name string, resolvedVal *string, commit bool) BuildStep {
	return &ArgStep{
		baseStep:    newBaseStep(Arg, args, commit),
		name:        name,
		resolvedVal: resolvedVal,
	}
}

// UpdateCtxAndConfig updates mutable states in build context, and generates a
// new image config base on config from previous step.
func (s *ArgStep) UpdateCtxAndConfig(
	ctx *context.BuildContext, imageConfig *image.Config) (*image.Config, error) {

	// Update in-memory map of merged stage vars from ARG and ENV.
	if s.resolvedVal != nil {
		ctx.StageVars[s.name] = *s.resolvedVal
	}

	return image.NewImageConfigFromCopy(imageConfig)
}
