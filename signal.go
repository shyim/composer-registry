//go:build !windows
// +build !windows

package main

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
)

func registerSignalHandlers() {
	handlePackageReload()
	handleConfigReload()
}

func handleConfigReload() {
	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGUSR1)

	go func() {
		for range sigs {
			log.Infof("Received signal, reloading config")
			newConfig, err := LoadConfig()
			if err != nil {
				log.Errorf("Failed to reload config: %s", err)
				continue
			}

			config = newConfig
		}
	}()
}

func handlePackageReload() {
	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGUSR2)

	go func() {
		for range sigs {
			log.Infof("Received signal, updating all packages")
			updateAll(true)
		}
	}()
}
