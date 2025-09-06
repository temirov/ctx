package main

import (
	"log"

	"github.com/temirov/ctx/internal/cli"
	"github.com/temirov/ctx/internal/utils"
)

// main is the entry point for installing the ctx command at the module root.
func main() {
	if executionError := cli.Execute(); executionError != nil {
		log.Fatalf(utils.ErrorLogFormat, executionError)
	}
}
