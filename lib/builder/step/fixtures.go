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

// +build !bins

package step

import (
	"fmt"
	"os"
)

var currUID int
var currGID int
var validChown string

func init() {
	currUID = os.Geteuid()
	currGID = os.Getegid()
	validChown = fmt.Sprintf("%d:%d", currUID, currGID)
}

// FromStepFixture returns a FromStep, panicing if it fails, for testing purposes.
func FromStepFixture(args, image, alias string) *FromStep {
	f, err := NewFromStep("", image, alias)
	if err != nil {
		panic(err)
	}
	return f
}

// AddStepFixture returns a AddStep, panicing if it fails, for testing purposes.
func AddStepFixture(args string, srcs []string, dst string, commit bool) *AddStep {
	c, err := NewAddStep(args, validChown, srcs, dst, commit)
	if err != nil {
		panic(err)
	}
	return c
}

// AddStepFixtureNoChown returns a AddStep, panicing if it fails, for testing purposes.
func AddStepFixtureNoChown(args string, srcs []string, dst string, commit bool) *AddStep {
	c, err := NewAddStep(args, "", srcs, dst, commit)
	if err != nil {
		panic(err)
	}
	return c
}

// CopyStepFixture returns a CopyStep, panicing if it fails, for testing purposes.
func CopyStepFixture(args, fromStage string, srcs []string, dst string, commit bool) *CopyStep {
	c, err := NewCopyStep(args, validChown, fromStage, srcs, dst, commit)
	if err != nil {
		panic(err)
	}
	return c
}

// CopyStepFixtureNoChown returns a CopyStep, panicing if it fails, for testing purposes.
func CopyStepFixtureNoChown(args, fromStage string, srcs []string, dst string, commit bool) *CopyStep {
	c, err := NewCopyStep(args, "", fromStage, srcs, dst, commit)
	if err != nil {
		panic(err)
	}
	return c
}
