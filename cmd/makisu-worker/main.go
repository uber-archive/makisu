package main

import (
	"log"
	"os"

	"github.com/uber/makisu/cli"
	"github.com/uber/makisu/lib/utils"

	"github.com/apourchet/commander"
)

func main() {
	log.Printf("Starting makisu (version=%s)", utils.BuildHash)

	application := cli.NewWorkerApplication()
	cmd := commander.New()
	if err := cmd.RunCLI(application, os.Args[1:]); err != nil {
		log.Fatalf("Failed to run command: %s", err)
	}
}
