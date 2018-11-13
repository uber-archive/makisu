package cli

import (
	"fmt"
	"os"
	"runtime/pprof"

	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/storage"
	"github.com/uber/makisu/lib/utils"
	"github.com/apourchet/commander"

	"go.uber.org/zap"
)

// Application contains all CLI commands.
type Application struct {
	BuildFlags  `commander:"flagstruct=build"`
	ListenFlags `commander:"flagstruct=listen"`

	ClientCmd *ClientCommand `commander:"subcommand=client"`

	LogOutput string `commander:"flag=log-output,The output file path for the logs."`
	LogLevel  string `commander:"flag=log-level,The level at which to log."`
	LogFormat string `commander:"flag=log-fmt,The format of the logs."`

	Profile bool `commander:"flag=cpu-profile,Profile the application."`

	cleanups []func() error
}

// NewApplication returns a new instance of application. The version of the application is used as
// a seed to any cache ID computation, so different versions will result in a break of caching layers.
func NewApplication() (*Application, error) {
	app := &Application{
		BuildFlags:  newBuildFlags(),
		ListenFlags: newListenFlags(),
		ClientCmd:   newClientCommand(),
		LogOutput:   "stdout",
		LogLevel:    "info",
		LogFormat:   "json",
	}
	return app, nil
}

// CLIName returns the name of the application. Commander uses this when showing usage information.
func (app *Application) CLIName() string {
	return "makisu"
}

// Cleanup cleans up the application after it has run its command.
func (app *Application) Cleanup() error {
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
func (app *Application) PostFlagParse() error {
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
func (app *Application) AddCleanup(fn func() error) {
	app.cleanups = append(app.cleanups, fn)
}

// GetCommandDescription returns the description of the given application command. This
// is used by commander when displaying the help messages.
func (app *Application) GetCommandDescription(cmd string) string {
	switch cmd {
	case "build":
		return `Builds a docker image from a build context and a dockerfile.`
	case "listen":
		return `Instruct makisu to listen on a unix socket in the background for build requests like a daemon.`
	}
	return ""
}

// Help displays the usage of makisu.
func (app *Application) Help() {
	commander.New().PrintUsage(app, "makisu")
}

func (app *Application) getLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	if app.LogOutput != "stdout" {
		config.OutputPaths = []string{app.LogOutput}
	}

	if err := config.Level.UnmarshalText([]byte(app.LogLevel)); err != nil {
		return nil, fmt.Errorf("parse log level: %s", err)
	}

	config.Encoding = app.LogFormat
	config.DisableStacktrace = true
	return config.Build()
}

func (app *Application) setupProfiler() error {
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
