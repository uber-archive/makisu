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

package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/apourchet/commander"
	"github.com/uber/makisu/lib/client"
	"github.com/uber/makisu/lib/fileio"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/utils"
)

// ClientApplication is the subcommand for interacting with a makisu worker listening on a unix socket.
type ClientApplication struct {
	BuildFlags `commander:"flagstruct=build"`

	SocketPath       string `commander:"flag=s,The absolute path of the unix socket that the makisu worker listens on"`
	LocalSharedPath  string `commander:"flag=l,The absolute path of the local mountpoint shared with the makisu worker"`
	WorkerSharedPath string `commander:"flag=w,The absolute destination of the mountpoint shared with the makisu worker"`
	Exit             bool   `commander:"flag=exit,Whether the worker should exit after the build finishes"`

	cli *client.MakisuClient
}

// NewClientApplication returns a new client application for the build command. The client
// will talk to the makisu worker through the unix socket that is shared between the local
// fs and that of the worker container.
func NewClientApplication() *ClientApplication {
	return &ClientApplication{
		BuildFlags:       newBuildFlags(),
		SocketPath:       "/makisu-socket/makisu.sock",
		LocalSharedPath:  "/makisu-context",
		WorkerSharedPath: "/makisu-context",
	}
}

// PostFlagParse gets executed once the CLI flags have been parsed into the ClientCommand.
func (cmd *ClientApplication) PostFlagParse() error {
	cmd.cli = client.New(cmd.SocketPath, cmd.LocalSharedPath, cmd.WorkerSharedPath)
	cmd.cli.SetWorkerLog(func(line string) {
		fmt.Fprintf(os.Stderr, line+"\n")
	})
	return nil
}

// Ready returns an error if the worker is not ready for builds.
func (cmd *ClientApplication) Ready() error {
	if ready, err := cmd.cli.Ready(); err != nil {
		return err
	} else if !ready {
		return fmt.Errorf("worker not ready")
	}
	log.Infof("Worker is ready")
	return nil
}

// Build starts a build on the worker after copying the context over to it.
func (cmd *ClientApplication) Build(context string) error {
	defer func() {
		if cmd.Exit {
			log.Infof("Telling Makisu worker to exit")
			if err := cmd.cli.Exit(); err != nil {
				log.Errorf("Failed to tell worker to exit: %v", err)
			}
		}
	}()
	if err := cmd.placeDockerfile(context); err != nil {
		return fmt.Errorf("failed to move dockerfile into worker context: %v", err)
	}
	cmd.DockerfilePath = ".makisu.dockerfile"
	flags, err := commander.New().GetFlagSet(cmd.BuildFlags, "makisu build")
	if err != nil {
		return err
	}
	args := flags.Stringify()
	return cmd.cli.Build(args, context)
}

func (cmd *ClientApplication) placeDockerfile(context string) error {
	uid, gid, err := utils.GetUIDGID()
	if err != nil {
		return fmt.Errorf("failed to get uid and gid for dockerfile move: %v", err)
	}
	dest := filepath.Join(context, ".makisu.dockerfile")
	return fileio.NewCopier(nil).CopyFile(cmd.DockerfilePath, dest, uid, gid)
}
