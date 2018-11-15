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

package testutil

import (
	"os/user"
	"strconv"
)

// Cleanup contains a list of function that are called to cleanup a fixture
type Cleanup struct {
	funcs []func()
}

// Add adds function to funcs list
func (c *Cleanup) Add(f ...func()) {
	c.funcs = append(c.funcs, f...)
}

// AppendFront append funcs from another cleanup in front of the funcs list
func (c *Cleanup) AppendFront(c1 *Cleanup) {
	c.funcs = append(c1.funcs, c.funcs...)
}

// Recover runs cleanup functions after test exit with exception
func (c *Cleanup) Recover() {
	if err := recover(); err != nil {
		c.run()
		panic(err)
	}
}

// Run runs cleanup functions when a test finishes running
func (c *Cleanup) Run() {
	c.run()
}

func (c *Cleanup) run() {
	for _, f := range c.funcs {
		f()
	}
}

// CurrUser returns the string representation of the current user, used for
// testing only.
func CurrUser() string {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	return u.Username
}

// CurrUID returns the UID of the current user, used for testing only.
func CurrUID() int {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		panic(err)
	}
	return uid
}

// CurrGID returns the primary GID of the current user, used for testing only.
func CurrGID() int {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	uid, err := strconv.Atoi(u.Gid)
	if err != nil {
		panic(err)
	}
	return uid
}
