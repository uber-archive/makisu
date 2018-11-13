package main

import (
	"log"
	"os"

	"github.com/uber/makisu/lib/utils"

	"github.com/apourchet/commander"
	"github.com/uber/makisu/cli"
)

func main() {
	log.Printf("Starting makisu (version=%s)", utils.BuildHash)

	application, err := cli.NewApplication()
	if err != nil {
		log.Fatalf("Failed to init application: %s", err)
	}

	cmd := commander.New()
	if err := cmd.RunCLI(application, os.Args[1:]); err != nil {
		log.Fatalf("Failed to run command: %s", err)
	}

	if err := application.Cleanup(); err != nil {
		log.Fatalf("Failed to cleanup: %s", err)
	}
}
