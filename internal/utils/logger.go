package utils

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewApplicationLogger constructs a zap logger configured for human-readable console output.
func NewApplicationLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.Encoding = "console"
	config.DisableCaller = true
	config.DisableStacktrace = true
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	config.EncoderConfig.TimeKey = ""
	config.EncoderConfig.LevelKey = ""
	config.EncoderConfig.NameKey = ""
	config.EncoderConfig.CallerKey = ""
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.StacktraceKey = ""
	return config.Build()
}
