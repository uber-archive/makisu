package main

import (
	"log"
	"os"

	"github.com/uber/makisu/cli"

	"github.com/apourchet/commander"
)

func main() {
	application := cli.NewClientApplication()
	cmd := commander.New()
	if err := cmd.RunCLI(application, os.Args[1:]); err != nil {
		log.Fatalf("Failed to run command: %s", err)
	}
}
