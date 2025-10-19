package main

import (
	"fmt"

	"github.com/temirov/ctx/internal/cli"
	"github.com/temirov/ctx/internal/utils"
)

// main is the entry point for the ctx command.
func main() {
	loggerInstance, loggerInitializationError := utils.NewApplicationLogger()
	if loggerInitializationError != nil {
		panic(fmt.Errorf(utils.LoggerInitializationFailedMessageFormat, loggerInitializationError))
	}
	defer loggerInstance.Sync()
	if applicationExecutionError := cli.Execute(); applicationExecutionError != nil {
		loggerInstance.Fatal(utils.ApplicationExecutionFailedMessage + ": " + applicationExecutionError.Error())
	}
}
