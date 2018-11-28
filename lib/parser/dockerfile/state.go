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

// Stage represents a parsed dockerfile stage.
type Stage struct {
	From       *FromDirective
	Directives []Directive
}

// Stages is an alias for []*Stage.
type Stages []*Stage

func newStage(from *FromDirective) *Stage {
	return &Stage{from, make([]Directive, 0)}
}

func (s *Stage) addDirective(d Directive) {
	s.Directives = append(s.Directives, d)
}

// parsingState stores the parsing state and build stages of a dockerfile.
type parsingState struct {
	stages []*Stage

	// passedArgs contains the arguments passed in at runtime, used to
	// resolve the variables declared by ARG directives. The map should
	// not be modified by any directive.
	passedArgs map[string]string

	// globalArgs contains the resolved values corresponding to ARG
	// directives that occur before the first stage (FROM directive),
	// used for variable replacement in FROM directives.
	globalArgs map[string]string

	// stageVars contains the resolved values corresponding to ARG and
	// ENV directives that occurred during the current stage, used in
	// variable replacements in other directives in the stage.
	stageVars map[string]string
}

// newParsingState initializes a blank slate parsingState to begin parsing a dockerfile.
func newParsingState(vars map[string]string) *parsingState {
	return &parsingState{
		make([]*Stage, 0), vars, make(map[string]string), nil,
	}
}

func (s *parsingState) currStage() (*Stage, error) {
	if len(s.stages) == 0 {
		return nil, errBeforeFirstFrom
	}
	return s.stages[len(s.stages)-1], nil
}

func (s *parsingState) addStage(stage *Stage) {
	s.stages = append(s.stages, stage)
}

// Add this command to the build stage.
func (s *parsingState) addToCurrStage(d Directive) error {
	stage, err := s.currStage()
	if err != nil {
		return err
	}
	stage.addDirective(d)
	return nil
}
