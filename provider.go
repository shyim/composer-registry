package main

import "net/http"

type TypeProvider interface {
	GetConfig() ConfigProvider
	UpdateAll() error
	Webhook(*http.Request) error
}

var providers = make(map[string]TypeProvider)

func registerProviders(config *Config) {
	for _, provider := range config.Providers {
		switch provider.Type {
		case "gitlab":
			providers[provider.Name] = NewGitlabProvider(provider)
		case "github":
			providers[provider.Name] = NewGithubProvider(provider)
		}
	}
}
