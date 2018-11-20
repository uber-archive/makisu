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
	"runtime/pprof"

	"github.com/apourchet/commander"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/storage"
	"github.com/uber/makisu/lib/utils"

	"go.uber.org/zap"
)

// BuildApplication contains the bindings for the `makisu build` command.
type BuildApplication struct {
	ApplicationFlags `commander:"flagstruct"`
	BuildFlags       `commander:"flagstruct=build"`

	cleanups []func() error
}

// ApplicationFlags contains all of the flags for the top level CLI app.
type ApplicationFlags struct {
	HelpFlag  bool   `commander:"flag=help,Display usage information for Makisu."`
	LogOutput string `commander:"flag=log-output,The output file path for the logs."`
	LogLevel  string `commander:"flag=log-level,The level at which to log."`
	LogFormat string `commander:"flag=log-fmt,The format of the logs."`

	Profile bool `commander:"flag=cpu-profile,Profile the application."`
}

// NewBuildApplication returns a new instance of application. The version of the application is used as
// a seed to any cache ID computation, so different versions will result in a break of caching layers.
func NewBuildApplication() *BuildApplication {
	app := &BuildApplication{
		BuildFlags:       newBuildFlags(),
		ApplicationFlags: defaultApplicationFlags(),
	}
	return app
}

// defaultApplicationFlags returns the default values for the application flags.
func defaultApplicationFlags() ApplicationFlags {
	return ApplicationFlags{
		LogOutput: "stdout",
		LogLevel:  "info",
		LogFormat: "json",
	}
}

// CLIName returns the name of the application. Commander uses this when showing usage information.
func (app *BuildApplication) CLIName() string {
	return "makisu"
}

// Cleanup cleans up the application after it has run its command.
func (app *BuildApplication) Cleanup() error {
	app.AddCleanup(func() error {
		return storage.CleanupSandbox(app.BuildFlags.StorageDir)
	})

	errs := utils.NewMultiErrors()
	for _, cleanup := range app.cleanups {
		if err := cleanup(); err != nil {
			errs.Add(err)
		}
	}
	return errs.Collect()
}

// PostFlagParse initializes the standard logger and sets up profiling if need be.
// This implements the commander.PostFlagParseHook
// interface, so this function will get called after the Application struct gets injected with the
// values taken from the flags and arguments of the CLI; but before any of the Build/Pull (etc...)
// functions get called.
func (app *BuildApplication) PostFlagParse() error {
	if app.HelpFlag {
		app.Help()
		return nil
	}

	logger, err := app.getLogger()
	if err != nil {
		return fmt.Errorf("build logger: %v", err)
	}
	log.SetLogger(logger.Sugar())

	if app.Profile {
		if err := app.setupProfiler(); err != nil {
			return fmt.Errorf("setup profiler: %v", err)
		}
	}

	if err := app.BuildFlags.postInit(); err != nil {
		return err
	}
	return nil
}

// AddCleanup adds a cleanup function to run after the application exits.
func (app *BuildApplication) AddCleanup(fn func() error) {
	app.cleanups = append(app.cleanups, fn)
}

// GetCommandDescription returns the description of the given application command. This
// is used by commander when displaying the help messages.
func (app *BuildApplication) GetCommandDescription(cmd string) string {
	switch cmd {
	case "build":
		return `Builds a docker image from a build context and a dockerfile.`
	case "help":
		return `Displays Makisu usage information.`
	case "version":
		return `Displays the version of the makisu binary in use.`
	}
	return ""
}

// Help displays the usage of makisu.
func (app *BuildApplication) Help() {
	// Print the usage on a new build application so that the right defaults
	// show up in the output.
	commander.New().PrintUsage(NewBuildApplication(), "makisu")
}

// Version displays the version of the makisu binary in use.
func (app *BuildApplication) Version() error {
	fmt.Println(utils.BuildHash)
	return nil
}

// CommanderDefault gets called automatically when no subcommand is invoked.
func (app *BuildApplication) CommanderDefault() error {
	return fmt.Errorf("Need to specify a command for makisu. One of 'build', 'help' or 'version'")
}

func (app *BuildApplication) getLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	if app.LogOutput != "stdout" {
		config.OutputPaths = []string{app.LogOutput}
	}

	if err := config.Level.UnmarshalText([]byte(app.LogLevel)); err != nil {
		return nil, fmt.Errorf("parse log level: %s", err)
	}

	config.Encoding = app.LogFormat
	config.DisableStacktrace = true
	config.DisableCaller = true
	return config.Build()
}

func (app *BuildApplication) setupProfiler() error {
	f, err := os.Create("/tmp/makisu.prof")
	if err != nil {
		return fmt.Errorf("create profile file: %s", err)
	}
	pprof.StartCPUProfile(f)
	app.AddCleanup(func() error {
		pprof.StopCPUProfile()
		return nil
	})
	return nil
}
