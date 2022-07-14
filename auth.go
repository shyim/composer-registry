package main

import (
	"net/http"
	"strings"
)

func validateRequest(r *http.Request) *ConfigUser {
	if len(config.Users) == 0 {
		return &ConfigUser{}
	}

	token := strings.TrimPrefix(r.Header.Get("authorization"), "Bearer ")

	var found *ConfigUser
	for _, user := range config.Users {
		if token == user.Token {
			found = &user
			break
		}
	}

	return found
}
