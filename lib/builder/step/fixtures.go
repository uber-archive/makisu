// +build !bins

package step

import (
	"fmt"
	"os/user"
	"strconv"
)

var currUser string
var currUID int
var currGroup string
var currGID int
var validChown string

func init() {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	currUID, err = strconv.Atoi(u.Uid)
	if err != nil {
		panic(err)
	}
	currUser = u.Name
	g, err := user.LookupGroupId(u.Gid)
	if err != nil {
		panic(err)
	}
	currGID, err = strconv.Atoi(u.Gid)
	if err != nil {
		panic(err)
	}
	currGroup = g.Name
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
