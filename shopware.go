package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

type ShopwareProvider struct {
	provider ConfigProvider
}

func NewShopwareProvider(provider ConfigProvider) ShopwareProvider {
	return ShopwareProvider{provider: provider}
}

func (s ShopwareProvider) GetConfig() ConfigProvider {
	return s.provider
}

func (s ShopwareProvider) UpdateAll() error {
	return db.Batch(func(tx *bolt.Tx) error {
		for _, project := range s.provider.Projects {
			if err := s.updatePackages(tx, context.Background(), project.Name); err != nil {
				log.Errorf("cannot update all packages %s", err)
			}
		}

		return nil
	})
}

func (ShopwareProvider) Webhook(request *http.Request) error {
	return nil
}

func (s ShopwareProvider) RegisterCustomHTTPHandlers(router *httprouter.Router) {
	router.GET("/shopware/:owner/:repo/:version/file.zip", s.handleDownload)
}

func (s ShopwareProvider) updatePackages(tx *bolt.Tx, ctx context.Context, token string) error {
	r, _ := http.NewRequestWithContext(ctx, "GET", "https://packages.shopware.com/packages.json", nil)

	r.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(r)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	var response ComposerResponse

	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	for name, pkg := range response.Packages {
		for version, info := range pkg {
			dist := info["dist"].(map[string]interface{})

			if err := s.storeZip(ctx, name, version, dist["url"].(string), token); err != nil {
				log.Errorf("cannot download remote package (%s in version %s): %s", name, version, err)
			}

			link := fmt.Sprintf("%s/shopware/%s/%s/file.zip", config.URL, name, version)

			if err := addOrUpdateVersionDirect(tx, info, link, version, name+version); err != nil {
				log.Errorf("cannot update version %s:%s\n", name, version)
			}

			log.Infof("updated package %s in version %s", name, version)
		}
	}

	return nil
}

func (s ShopwareProvider) storeZip(ctx context.Context, name string, version string, url, token string) error {
	zipPath := getZipPath(name, version)
	zipFolder := filepath.Dir(zipPath)

	if _, err := os.Stat(zipPath); err == nil {
		return nil
	}

	r, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	r.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(r)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	if _, err := os.Stat(zipFolder); os.IsNotExist(err) {
		if err := os.MkdirAll(zipFolder, os.ModePerm); err != nil {
			return err
		}
	}

	return ioutil.WriteFile(zipPath, body, os.ModePerm)
}

func (s ShopwareProvider) handleDownload(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	user := validateRequest(r)

	if user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	packageName := fmt.Sprintf("%s/%s", ps.ByName("owner"), ps.ByName("repo"))

	zipFile := getZipPath(packageName, ps.ByName("version"))

	http.ServeFile(w, r, zipFile)
}

type ComposerResponse struct {
	Packages map[string]map[string]map[string]interface{} `json:"packages"`
}
