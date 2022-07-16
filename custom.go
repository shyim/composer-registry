package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type CustomProvider struct {
	provider ConfigProvider
}

func NewCustomProvider(provider ConfigProvider) CustomProvider {
	return CustomProvider{provider: provider}
}

func (c CustomProvider) GetConfig() ConfigProvider {
	return c.provider
}

func (c CustomProvider) UpdateAll() error {
	return nil
}

func (c CustomProvider) Webhook(r *http.Request) error {
	return nil
}

func (c CustomProvider) RegisterCustomHTTPHandlers(router *httprouter.Router) {
	router.POST("/custom/package/create", c.CreateVersion)
	router.DELETE("/custom/package/:owner/:repo/:version", c.DeleteVersion)
}

func (c CustomProvider) CreateVersion(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if !c.auth(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	zipReader, err := zip.NewReader(bytes.NewReader(reqBody), int64(len(reqBody)))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	var composerFileContent []byte
	for _, zipFile := range zipReader.File {
		if filepath.Base(zipFile.Name) == "composer.json" {
			f, err := zipFile.Open()
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				return
			}

			defer f.Close()

			fileContent, err := ioutil.ReadAll(f)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				return
			}

			composerFileContent = fileContent
			break
		}
	}

	if len(composerFileContent) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Cannot find composer.json in ZIP file."))
		return
	}

	composerJson := map[string]interface{}{}
	err = json.Unmarshal(composerFileContent, &composerJson)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	packageName, ok := composerJson["name"].(string)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Cannot find package name in composer.json."))
		return
	}

	packageVersion, ok := composerJson["version"].(string)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Cannot find package version in composer.json."))
		return
	}

	link := fmt.Sprintf("%s/custom/%s/%s/file.zip", config.URL, packageName, packageVersion)

	err = db.Update(func(tx *bolt.Tx) error {
		return addOrUpdateVersionDirect(tx, composerJson, link, packageVersion, "custom-"+packageName+"-"+packageVersion)
	})

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	zipPath := getZipPath(packageName, packageVersion)
	zipFolder := filepath.Dir(zipPath)

	if _, err := os.Stat(zipFolder); os.IsNotExist(err) {
		if err := os.MkdirAll(zipFolder, os.ModePerm); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
	}

	if err := ioutil.WriteFile(zipPath, reqBody, os.ModePerm); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c CustomProvider) DeleteVersion(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if !c.auth(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	owner := ps.ByName("owner")
	repo := ps.ByName("repo")
	version := ps.ByName("version")

	key := fmt.Sprintf("custom-%s/%s-%s", owner, repo, version)

	log.Infof(key)

	err := db.Update(func(tx *bolt.Tx) error {
		return deleteVersion(tx, key)
	})

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	zipPath := getZipPath(owner+"/"+repo, version)
	if err := os.Remove(zipPath); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c CustomProvider) auth(r *http.Request) bool {
	if c.provider.WebhookSecret == "" {
		return true
	}

	authHeader := strings.ToLower(r.Header.Get("authorization"))

	return strings.TrimPrefix(authHeader, "bearer ") == c.provider.WebhookSecret
}
