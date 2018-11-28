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
	"fmt"
	"strconv"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
)

// StopsignalStep implements BuildStep and execute STOPSIGNAL directive.
type StopsignalStep struct {
	*baseStep

	Signal int
}

// NewStopsignalStep returns a BuildStep from given arguments.
func NewStopsignalStep(args string, signal int, commit bool) BuildStep {
	return &StopsignalStep{
		baseStep: newBaseStep(Stopsignal, args, commit),
		Signal:   signal,
	}
}

// UpdateCtxAndConfig updates mutable states in build context, and generates a
// new image config base on config from previous step.
func (s *StopsignalStep) UpdateCtxAndConfig(
	ctx *context.BuildContext, imageConfig *image.Config) (*image.Config, error) {

	config, err := image.NewImageConfigFromCopy(imageConfig)
	if err != nil {
		return nil, fmt.Errorf("copy image config: %s", err)
	}
	config.Config.StopSignal = strconv.Itoa(s.Signal)
	return config, nil
}
