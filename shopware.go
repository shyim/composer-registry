package main

import (
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"io/ioutil"
	"net/http"
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
	//for _, project := range s.provider.Projects {
	//	if err := s.updatePackages(context.Background(), project.Name); err != nil {
	//		log.Errorf("cannot update all packages %s", err)
	//	}
	//}

	return nil
}

func (ShopwareProvider) Webhook(request *http.Request) error {
	return nil
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
			link := fmt.Sprintf("%s/shopware/%s/%s.zip", config.URL, name, version)
			if err := addOrUpdateVersionDirect(tx, info, link, version); err != nil {
				log.Errorf("cannot update version %s:%s\n", name, version)
			}
		}
	}

	return nil
}

type ComposerResponse struct {
	Packages map[string]map[string]map[string]interface{} `json:"packages"`
}
