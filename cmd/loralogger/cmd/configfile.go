package cmd

import (
	"os"
	"text/template"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/rbricheno/lora-logger-dynamodb/internal/config"
)

const configTemplate = `[general]
# Log level
#
# debug=5, info=4, warning=3, error=2, fatal=1, panic=0
log_level={{ .General.LogLevel }}


[loralogger]
# Bind
#
# The interface:port on which the lora-logger will bind for receiving
# data from the packet-forwarder (UDP data).
bind="{{ .LoraLogger.Bind }}"

# Region
#
# The region in which the DynamoDB is running.
region="{{ .LoraLogger.Region }}"

# Table
#
# The name of the DynamoDB table to use.
table="{{ .LoraLogger.Table }}"

# Credentials path
#
# The path to your AWS shared credentials.
credentials_path="{{ .LoraLogger.CredentialsPath }}"

# Credentials profile
#
# The profile from your AWS shared credentials.
credentials_profile="{{ .LoraLogger.CredentialsProfile }}"
`

var configCmd = &cobra.Command{
	Use:   "configfile",
	Short: "Print the LoRa Server configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		t := template.Must(template.New("config").Parse(configTemplate))
		err := t.Execute(os.Stdout, &config.C)
		if err != nil {
			return errors.Wrap(err, "execute config template error")
		}
		return nil
	},
}
