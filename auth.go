package main

import (
	"net/http"
	"strings"
)

func validateRequest(r *http.Request) *ConfigUser {
	if len(config.Users) == 0 {
		return &ConfigUser{Rules: make([]ConfigUserRule, 0)}
	}

	token := strings.TrimPrefix(strings.ToLower(r.Header.Get("authorization")), "bearer ")

	var found *ConfigUser
	for _, user := range config.Users {
		if token == user.Token {
			found = &user
			break
		}
	}

	return found
}
