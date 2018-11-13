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
