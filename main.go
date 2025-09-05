package main

import (
	"log"

	"github.com/temirov/ctx/internal/cli"
)

const errorLogFormat = "Error: %v"

// main is the entry point for the ctx application.
func main() {
	if err := cli.Execute(); err != nil {
		log.Fatalf(errorLogFormat, err)
	}
}
