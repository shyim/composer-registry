package main

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

func validateRequest(r *http.Request) *ConfigUser {
	if len(config.Users) == 0 {
		return &ConfigUser{Rules: make([]ConfigUserRule, 0)}
	}

	token := []byte(strings.TrimPrefix(strings.TrimPrefix(r.Header.Get("authorization"), "bearer "), "Bearer "))

	var found *ConfigUser
	for _, user := range config.Users {
		if subtle.ConstantTimeCompare(token, []byte(user.Token)) == 1 {
			found = &user
			break
		}
	}

	return found
}
