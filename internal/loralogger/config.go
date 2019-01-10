package loralogger

// Config holds the loralogger config.
type Config struct {
	Bind   string `mapstructure:"bind"`
	Region string `mapstructure:"region"`
	Table  string `mapstructure:"table"`
}
