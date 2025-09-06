package main

import (
	"log"

	"github.com/temirov/ctx/internal/cli"
	"github.com/temirov/ctx/internal/utils"
)

// main is the entry point when building the ctx command from the cmd/ctx path.
func main() {
	if executionError := cli.Execute(); executionError != nil {
		log.Fatalf(utils.ErrorLogFormat, executionError)
	}
}
