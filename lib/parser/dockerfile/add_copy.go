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
	"strings"
)

type addCopyDirective struct {
	*baseDirective
	Chown         string
	PreserveOwner bool
	Srcs          []string
	Dst           string
}

// Variables:
//   Replaced from ARGs and ENVs from within our stage.
// Formats:
//   ADD/COPY [--archive] <src>... <dest>
//   ADD/COPY [--archive] ["<src>",... "<dest>"]
//   ADD/COPY [--chown=<user>:<group>] ["<src>",... "<dest>"]
//   ADD/COPY [--chown=<user>:<group>] <src>... <dest>
func newAddCopyDirective(base *baseDirective, args []string) (*addCopyDirective, error) {
	if len(args) == 0 {
		return nil, base.err(errMissingArgs)
	}

	// Check the flag numbers here since we only allow zero or one flag here.
	var chownCount, archiveCount int
	var chown string
	var preserveOwner bool
	for _, arg := range args[:len(args)-1] {
		if strings.HasPrefix(arg, "--chown") {
			if val, ok, err := parseStringFlag(arg, "chown"); err != nil {
				return nil, base.err(err)
			} else if ok {
				chown = val
				chownCount++
				continue
			}
		}

		if strings.HasPrefix(arg, "--archive") {
			if err := parseBoolFlag(arg, "archive"); err == nil {
				archiveCount++
				preserveOwner = true
			} else {
				return nil, fmt.Errorf("archive flag format is wrong")
			}
		}
	}

	if archiveCount+chownCount >= 2 {
		return nil, base.err(fmt.Errorf("argument shouldn't contain more than one flag [--chown or --archive]"))
	} else if archiveCount+chownCount == 1 {
		args = args[1:]
	}

	var parsed []string
	if json, ok := parseJSONArray(strings.Join(args, " ")); ok {
		parsed = json
	} else {
		parsed = args
	}
	if len(parsed) < 2 {
		return nil, base.err(errMissingArgs)
	}
	srcs := parsed[:len(parsed)-1]
	dst := parsed[len(parsed)-1]
	return &addCopyDirective{base, chown, preserveOwner, srcs, dst}, nil
}
