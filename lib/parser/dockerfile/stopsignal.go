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

package dockerfile

import (
	"fmt"
	"strconv"
)

// StopsignalDirective represents the "STOPSIGNAL" dockerfile command.
type StopsignalDirective struct {
	*baseDirective
	Signal int
}

// Formats:
//   STOPSIGNAL <value> ...
func newStopsignalDirective(base *baseDirective, state *parsingState) (Directive, error) {
	signal, err := strconv.Atoi(base.Args)
	if err != nil {
		return nil, fmt.Errorf("signal must be integer: %s", err)
	} else if signal < 0 {
		return nil, fmt.Errorf("signal must be > 0: %v", signal)
	}
	return &StopsignalDirective{base, signal}, nil
}

// Add this command to the build stage.
func (d *StopsignalDirective) update(state *parsingState) error {
	return state.addToCurrStage(d)
}
