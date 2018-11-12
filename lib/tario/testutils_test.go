package tario

import (
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/shell"
)

func untarHelper(t, out string) error {
	// Use p option to preserve permissions.
	return shell.ExecCommand(log.Infof, log.Errorf, "", "tar", "xfp", t, "-C", out)
}
