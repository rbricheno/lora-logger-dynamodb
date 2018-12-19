package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/rbricheno/lora-logger/internal/config"
	"github.com/rbricheno/lora-logger/internal/loralogger"
)

func run(cmd *cobra.Command, args []string) error {
	m, err := loralogger.New(config.C.LoraLogger)
	if err != nil {
		return errors.Wrap(err, "new loralogger error")
	}

	sigChan := make(chan os.Signal)
	exitChan := make(chan struct{})
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	log.WithField("signal", <-sigChan).Info("signal received")
	go func() {
		log.Warning("stopping loralogger")
		if err := m.Close(); err != nil {
			log.Fatal(err)
		}
		exitChan <- struct{}{}
	}()
	select {
	case <-exitChan:
	case s := <-sigChan:
		log.WithField("signal", s).Info("signal received, stopping immediately")
	}

	return nil
}
