package dockerfile

import (
	"bufio"
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
	for scanner.Scan() {
		if directive, err := newDirective(scanner.Text(), state); err != nil {
			return nil, err
		} else if directive == nil {
			continue
		} else if err := directive.update(state); err != nil {
			return nil, err
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return state.stages, nil
}
