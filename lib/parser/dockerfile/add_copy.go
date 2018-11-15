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

import "strings"

type addCopyDirective struct {
	*baseDirective
	Chown string
	Srcs  []string
	Dst   string
}

// Variables:
//   Replaced from ARGs and ENVs from within our stage.
// Formats:
//   ADD/COPY [--chown=<user>:<group>] ["<src>",... "<dest>"]
//   ADD/COPY [--chown=<user>:<group>] <src>... <dest>
func newAddCopyDirective(base *baseDirective, args []string) (*addCopyDirective, error) {
	if len(args) == 0 {
		return nil, base.err(errMissingArgs)
	}

	var chown string
	if val, ok, err := parseFlag(args[0], "chown"); err != nil {
		return nil, base.err(err)
	} else if ok {
		chown = val
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

	return &addCopyDirective{base, chown, srcs, dst}, nil
}
