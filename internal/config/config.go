package config

import "github.com/rbricheno/lora-logger/internal/loralogger"

// Version holds the LoRa Logger version.
var Version string

// Config defines the configuration structure.
type Config struct {
	General struct {
		LogLevel int `mapstructure:"log_level"`
	} `mapstructure:"general"`

	LoraLogger loralogger.Config `mapstructure:"loralogger"`
}

// C holds the configuration.
var C Config
