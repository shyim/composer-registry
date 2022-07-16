package main

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
)

type ConfigUser struct {
	Token string `yaml:"token"`
}

type Config struct {
	Providers   []ConfigProvider `yaml:"providers"`
	Users       []ConfigUser     `yaml:"users"`
	URL         string           `yaml:"base_url"`
	StoragePath string           `yaml:"storage_path"`
}
type ConfigProjects struct {
	Name string `yaml:"name"`
}
type ConfigProvider struct {
	Name            string           `yaml:"name"`
	Type            string           `yaml:"type"`
	Domain          string           `yaml:"domain"`
	Token           string           `yaml:"token"`
	WebhookSecret   string           `yaml:"webhook_secret"`
	Projects        []ConfigProjects `yaml:"projects"`
	FetchAllOnStart bool             `yaml:"fetch_all_on_start"`
}

func LoadConfig() (*Config, error) {
	bytes, err := ioutil.ReadFile("config.yml")

	if err != nil {
		return nil, err
	}

	var config Config

	err = yaml.Unmarshal(bytes, &config)

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
