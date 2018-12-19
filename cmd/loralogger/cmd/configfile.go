package cmd

import (
	"os"
	"text/template"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/rbricheno/lora-logger/internal/config"
)

const configTemplate = `[general]
# Log level
#
# debug=5, info=4, warning=3, error=2, fatal=1, panic=0
log_level={{ .General.LogLevel }}


[lora_logger]
# Bind
#
# The interface:port on which the lora-logger will bind for receiving
# data from the packet-forwarder (UDP data).
bind="{{ .LoraLogger.Bind }}"
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
