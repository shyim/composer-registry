package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v11"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type ConfigUser struct {
	Token string           `yaml:"token" json:"token"`
	Rules []ConfigUserRule `yaml:"rules" json:"rules"`
}

type ConfigUserRule struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type Config struct {
	Providers   []ConfigProvider `yaml:"providers" json:"providers"`
	Users       []ConfigUser     `yaml:"users" json:"users"`
	URL         string           `yaml:"base_url" json:"base_url" env:"COMPOSER_REGISTRY_URL"`
	StoragePath string           `yaml:"storage_path" json:"storage_path" env:"COMPOSER_REGISTRY_STORAGE_PATH"`
	BindAddress string           `yaml:"bind_address" json:"bind_address" env:"COMPOSER_REGISTRY_BIND_ADDRESS"`
}
type ConfigProjects struct {
	Name string `yaml:"name"`
}
type ConfigProvider struct {
	Name            string           `yaml:"name" json:"name"`
	Type            string           `yaml:"type" json:"type"`
	Domain          string           `yaml:"domain" json:"domain"`
	Token           string           `yaml:"token" json:"token"`
	WebhookSecret   string           `yaml:"webhook_secret" json:"webhook_secret"`
	Projects        []ConfigProjects `yaml:"projects" json:"projects"`
	FetchAllOnStart bool             `yaml:"fetch_all_on_start" json:"fetch_all_on_start"`
	CronSchedule    string           `yaml:"cron_schedule" json:"cron_schedule"`
}

func LoadConfig() (*Config, error) {
	var config Config

	if _, err := os.Stat(configFile); err != nil {
		return nil, fmt.Errorf("cannot find config file at %s", configFile)
	}

	bytes, err := os.ReadFile(configFile)

	if err != nil {
		return nil, err
	}

	configExtension := filepath.Ext(configFile)
	if configExtension == ".yml" || configExtension == ".yaml" {
		err = yaml.Unmarshal(bytes, &config)

		if err != nil {
			return nil, err
		}
	} else if configExtension == ".json" {
		err = json.Unmarshal(bytes, &config)

		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("config file is not a json or yaml file")
	}

	err = env.Parse(&config)
	if err != nil {
		return nil, err
	}

	if config.StoragePath == "" {
		cwd, err := os.Getwd()

		if err != nil {
			config.StoragePath = "storage"
		} else {
			config.StoragePath = path.Join(cwd, "storage")
		}
	}

	if config.BindAddress == "" {
		config.BindAddress = "127.0.0.1:8080"
	}

	if _, err := os.Stat(config.StoragePath); os.IsNotExist(err) {
		if err := os.MkdirAll(config.StoragePath, os.ModePerm); err != nil {
			return nil, err
		}
	}

	log.Infof("config: using %s as storage", config.StoragePath)

	if config.URL == "" {
		config.URL = "http://localhost:8080"
		log.Infof("config: base_url is not set in config. defaulting to http://localhost:8080")
	}

	return &config, nil
}

func getZipPath(name string, version string) string {
	return path.Join(config.StoragePath, "packages", name, version+".zip")
}

func (c ConfigUser) HasAccessToPackage(packageName string) bool {
	if len(c.Rules) == 0 {
		return true
	}

	for _, rule := range c.Rules {
		switch rule.Type {
		case "begins_with":
			if strings.HasPrefix(packageName, rule.Value) {
				return true
			}
		case "ends_with":
			if strings.HasSuffix(packageName, rule.Value) {
				return true
			}
		case "contains":
			if strings.Contains(packageName, rule.Value) {
				return true
			}
		case "equals":
			if packageName == rule.Value {
				return true
			}
		}
	}

	return false
}
