package dockerfile

import (
	"bufio"
	"fmt"
	"strings"
)

// ParseFile parses dockerfile from given reader, returns a ParsedFile object.
func ParseFile(filecontents string, args map[string]string) ([]*Stage, error) {
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
			return nil, fmt.Errorf("file scanning failed (line %d): %v", count, err)
		}
		text := scanner.Text()
		if directive, err := newDirective(text, state); err != nil {
			return nil, fmt.Errorf("failed to create new directive (line %d): %v", count, err)
		} else if directive == nil {
			continue
		} else if err := directive.update(state); err != nil {
			return nil, fmt.Errorf("failed to update parser state (line %d): %v", count, err)
		}
	}

	return state.stages, nil
}
