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
	"bufio"
	"fmt"
	"strings"
)

// ParseFile parses dockerfile from given reader, returns a ParsedFile object.
func ParseFile(filecontents string, args map[string]string) ([]*Stage, error) {
	filecontents = removeCommentLines(filecontents)
	filecontents = strings.Replace(filecontents, "\\\n", "", -1)
	reader := strings.NewReader(filecontents)
	scanner := bufio.NewScanner(reader)

	if args == nil {
		args = make(map[string]string)
	}

	state := newParsingState(args)
	var count int
	for scanner.Scan() {
		count++
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("file scanning failed (line %d): %s", count, err)
		}
		text := scanner.Text()
		if directive, err := newDirective(text, state); err != nil {
			return nil, fmt.Errorf("failed to create new directive (line %d): %s", count, err)
		} else if directive == nil {
			continue
		} else if err := directive.update(state); err != nil {
			return nil, fmt.Errorf("failed to update parser state (line %d): %s", count, err)
		}
	}

	return state.stages, nil
}

func removeCommentLines(filecontents string) string {
	lines := strings.Split(filecontents, "\n")
	var output string
	for _, line := range lines {
		trimmed := strings.Trim(line, " \t")
		if len(trimmed) != 0 && trimmed[0] == '#' {
			continue
		} else if len(trimmed) == 0 {
			continue
		}
		output += line + "\n"
	}
	return output
}
