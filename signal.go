package main

import (
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

func registerSignalHandlers() {
	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGUSR2)

	go func() {
		for range sigs {
			log.Infof("Received signal, updating all packages")
			updateAll(true)
		}
	}()
}
