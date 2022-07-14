package main

import "net/http"

var packages = make(map[string]map[string]interface{})

type TypeProvider interface {
	UpdateAll() error
	Webhook(*http.Request) error
}

var providers = make(map[string]TypeProvider)

func registerProviders(config *Config) {
	for _, provider := range config.Providers {
		switch provider.Type {
		case "gitlab":
			providers[provider.Name] = NewGitlabProvider(provider)
		}
	}
}
