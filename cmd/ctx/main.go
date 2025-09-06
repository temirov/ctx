package main

import (
	"fmt"

	"github.com/temirov/ctx/internal/cli"
	"github.com/temirov/ctx/internal/utils"
	"go.uber.org/zap"
)

// main is the entry point for the ctx command.
func main() {
	loggerInstance, loggerInitializationError := zap.NewProduction()
	if loggerInitializationError != nil {
		panic(fmt.Errorf(utils.LoggerInitializationFailedMessageFormat, loggerInitializationError))
	}
	defer loggerInstance.Sync()
	if applicationExecutionError := cli.Execute(); applicationExecutionError != nil {
		loggerInstance.Fatal(utils.ApplicationExecutionFailedMessage, zap.Error(applicationExecutionError))
	}
}
