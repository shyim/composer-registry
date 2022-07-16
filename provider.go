package main

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
)

type TypeProvider interface {
	GetConfig() ConfigProvider
	UpdateAll() error
	Webhook(*http.Request) error
	RegisterCustomHTTPHandlers(*httprouter.Router)
}

var providers = make(map[string]TypeProvider)

func registerProviders(config *Config, router *httprouter.Router) {
	registeredProviders := make(map[string]bool)

	for _, provider := range config.Providers {
		switch provider.Type {
		case "gitlab":
			providers[provider.Name] = NewGitlabProvider(provider)
		case "github":
			providers[provider.Name] = NewGithubProvider(provider)
		case "shopware":
			providers[provider.Name] = NewShopwareProvider(provider)
		}

		if _, ok := registeredProviders[provider.Type]; !ok {
			providers[provider.Name].RegisterCustomHTTPHandlers(router)

			registeredProviders[provider.Type] = true
		}
	}
}
