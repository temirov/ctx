package utils

const (
	// EmptyString represents a reusable empty string constant.
	EmptyString = ""
	// ErrorLogFormat defines the formatting string for error log messages.
	ErrorLogFormat = "Error: %v"
	// LoggerInitializationFailedMessageFormat defines the message format for logger initialization failures.
	LoggerInitializationFailedMessageFormat = "failed to initialize logger: %v"
	// ApplicationExecutionFailedMessage denotes the error message when application execution fails.
	ApplicationExecutionFailedMessage = "application execution failed"
	// ConfigFileBaseName is the base name for application configuration files.
	ConfigFileBaseName = "config"
	// ConfigFileExtension identifies the configuration file format.
	ConfigFileExtension = "yaml"
	// ConfigFileName is the default configuration file name.
	ConfigFileName = ConfigFileBaseName + "." + ConfigFileExtension
	// GlobalConfigDirectoryName is the directory under the user's home containing global configuration.
	GlobalConfigDirectoryName = ".ctx"
)
