package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type ConfigUser struct {
	Token string `yaml:"token"`
}

type Config struct {
	Providers []ConfigProvider `yaml:"providers"`
	Users     []ConfigUser     `yaml:"users"`
}
type ConfigProjects struct {
	Name string `yaml:"name"`
}
type ConfigProvider struct {
	Name          string           `yaml:"name"`
	Type          string           `yaml:"type"`
	Domain        string           `yaml:"domain"`
	Token         string           `yaml:"token"`
	WebhookSecret string           `yaml:"webhook_secret"`
	Projects      []ConfigProjects `yaml:"projects"`
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

	return &config, nil
}
