package main

import (
	"log"

	"github.com/temirov/ctx/internal/cli"
	"github.com/temirov/ctx/internal/utils"
)

// main is the entry point for the ctx command.
func main() {
	if err := cli.Execute(); err != nil {
		log.Fatalf(utils.ErrorLogFormat, err)
	}
}
