package cli

import (
	"fmt"
	"os"

	"github.com/uber/makisu/lib/client"
	"github.com/uber/makisu/lib/log"
	"github.com/apourchet/commander"
)

// ClientCommand is the subcommand for interacting with a makisu worker listening on a unix socket.
type ClientCommand struct {
	BuildFlags `commander:"flagstruct=build"`

	SocketPath       string `commander:"flag=s,The absolute path of the unix socket that the makisu worker listens on"`
	LocalSharedPath  string `commander:"l,The absolute path of the local mountpoint shared with the makisu worker"`
	WorkerSharedPath string `commander:"w,The absolute destination of the mountpoint shared with the makisu worker"`

	cli *client.MakisuClient
}

func newClientCommand() *ClientCommand {
	return &ClientCommand{
		BuildFlags:       newBuildFlags(),
		SocketPath:       "/makisu-socket/makisu.sock",
		LocalSharedPath:  "/makisu-context",
		WorkerSharedPath: "/makisu-context",
	}
}

// PostFlagParse gets executed once the CLI flags have been parsed into the ClientCommand.
func (cmd *ClientCommand) PostFlagParse() error {
	cmd.cli = client.New(cmd.SocketPath, cmd.LocalSharedPath, cmd.WorkerSharedPath)
	cmd.cli.SetWorkerLog(func(line string) {
		fmt.Fprintf(os.Stderr, line+"\n")
	})
	return nil
}

// Ready returns an error if the worker is not ready for builds.
func (cmd *ClientCommand) Ready() error {
	if ready, err := cmd.cli.Ready(); err != nil {
		return err
	} else if !ready {
		return fmt.Errorf("worker not ready")
	}
	log.Infof("Worker is ready")
	return nil
}

// Build starts a build on the worker after copying the context over to it.
func (cmd *ClientCommand) Build(context string) error {
	flags, err := commander.New().GetFlagSet(cmd.BuildFlags, "makisu build")
	if err != nil {
		return err
	}
	args := flags.Stringify()
	return cmd.cli.Build(args, context)
}
