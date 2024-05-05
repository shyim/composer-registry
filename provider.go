package main

import (
	"net/http"

	"github.com/go-co-op/gocron/v2"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

type TypeProvider interface {
	GetConfig() ConfigProvider
	UpdateAll() error
	Webhook(*http.Request) error
	RegisterCustomHTTPHandlers(*httprouter.Router)
}

var providers = make(map[string]TypeProvider)

func registerProviders(config *Config, router *httprouter.Router) {
	s, _ := gocron.NewScheduler()

	registeredProviders := make(map[string]bool)

	for _, provider := range config.Providers {
		switch provider.Type {
		case "gitlab":
			providers[provider.Name] = NewGitlabProvider(provider)
		case "github":
			providers[provider.Name] = NewGithubProvider(provider)
		case "shopware":
			providers[provider.Name] = NewShopwareProvider(provider)
		case "custom":
			providers[provider.Name] = NewCustomProvider(provider)
		}

		if _, ok := registeredProviders[provider.Type]; !ok {
			providers[provider.Name].RegisterCustomHTTPHandlers(router)

			registeredProviders[provider.Type] = true
		}

		if provider.CronSchedule != "" {
			s.NewJob(
				gocron.CronJob(provider.CronSchedule, false),
				gocron.NewTask(providers[provider.Name].UpdateAll),
				gocron.WithTags(provider.Name),
			)
		}
	}

	s.Start()
}

func updateAll(force bool) {
	for name, provider := range providers {
		if provider.GetConfig().FetchAllOnStart || force {
			log.Infof("Updating all packages of %s", name)
			if err := provider.UpdateAll(); err != nil {
				log.Infof("Error updating all packages of %s: %s", name, err)
			}
		}
	}
}
