package main

import (
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"net/http"
	"os"
	"strings"
)

var config *Config
var db *bolt.DB

const DB_PATH = "packages.db"

func main() {
	router := httprouter.New()
	router.GET("/packages.json", packagesJsonHandler)
	router.POST("/webhook/:name", webhookHandler)

	var err error
	config, err = LoadConfig()
	if err != nil {
		log.Fatalln(err)
	}

	db, err = bolt.Open(DB_PATH, 0666, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	registerProviders(config)

	go updateAll(false)

	bindAddress := os.Getenv("BIND_ADDRESS")

	if bindAddress == "" {
		bindAddress = "127.0.0.1:8080"
	}

	log.Fatal(http.ListenAndServe(bindAddress, router))
}

func webhookHandler(writer http.ResponseWriter, request *http.Request, ps httprouter.Params) {
	if request.Method != http.MethodPost {
		http.Error(writer, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	providerName := ps.ByName("name")

	if _, ok := providers[providerName]; !ok {
		http.Error(writer, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	log.Infof("Received webhook from %s", ps.ByName("name"))

	if err := providers[providerName].Webhook(request); err != nil {
		log.Infof("Webhook error: %s", err)
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	writer.WriteHeader(http.StatusOK)
}

func packagesJsonHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	token := strings.TrimPrefix(r.Header.Get("authorization"), "Bearer ")

	if len(config.Users) != 0 {
		found := false

		for _, user := range config.Users {
			if token == user.Token {
				found = true
				break
			}
		}

		if !found {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
	}

	var packages = make(map[string]map[string]interface{})

	db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("packages")).ForEach(func(k, v []byte) error {
			var composerJson map[string]interface{}

			if err := json.Unmarshal(v, &composerJson); err != nil {
				return err
			}

			packageNameSplit := strings.Split(string(k), "|")

			if _, ok := packages[packageNameSplit[0]]; !ok {
				packages[packageNameSplit[0]] = make(map[string]interface{})
			}

			packages[packageNameSplit[0]][packageNameSplit[1]] = composerJson

			return nil
		})
	})

	err := json.NewEncoder(w).Encode(map[string]interface{}{"packages": packages})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func addOrUpdate(tx *bolt.Tx, bytes []byte, version, downloadLink string) error {
	composerJson := map[string]interface{}{}

	if err := json.Unmarshal(bytes, &composerJson); err != nil {
		return err
	}

	packageName := composerJson["name"].(string)

	composerJson["dist"] = map[string]string{
		"url":  downloadLink,
		"type": "zip",
	}

	composerJson["version"] = version

	key := fmt.Sprintf("%s|%s", packageName, version)

	bucket, err := tx.CreateBucketIfNotExists([]byte("packages"))

	if err != nil {
		return err
	}

	composerJsonData, _ := json.Marshal(composerJson)

	return bucket.Put([]byte(key), composerJsonData)
}

func updateAll(force bool) {
	for name, provider := range providers {
		if provider.GetConfig().FetchAllOnStart || force {
			log.Infof("Updating all packages of %s", name)
			if err := provider.UpdateAll(); err != nil {
				log.Infof("Error updating all packages of %s: %s", name, err)
			}
		}
	}
}
