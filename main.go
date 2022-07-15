package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"net/http"
	"os"
	"sort"
	"strings"
)

var config *Config
var db *bolt.DB

const DB_PATH = "packages.db"

func main() {
	router := httprouter.New()
	router.GET("/packages.json", packagesJsonHandler)
	router.GET("/p/:owner/:repo/versions.json", singlePackageHandler)
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

	registerSignalHandlers()

	bindAddress := os.Getenv("BIND_ADDRESS")

	if bindAddress == "" {
		bindAddress = "127.0.0.1:8080"
	}

	log.Infof("Listing on %s", bindAddress)
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
	user := validateRequest(r)

	if user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	var availablePackagesIndexed = make(map[string]bool)

	err := db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("packages")).ForEach(func(k, v []byte) error {
			packageNameSplit := strings.Split(string(k), "|")
			availablePackagesIndexed[packageNameSplit[0]] = true

			return nil
		})
	})

	if err != nil {
		log.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var availablePackages []string
	for packageName := range availablePackagesIndexed {
		availablePackages = append(availablePackages, packageName)
	}

	sort.Strings(availablePackages)

	err = json.NewEncoder(w).Encode(map[string]interface{}{"metadata-url": "/p/%package%/versions.json", "available-packages": availablePackages})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func singlePackageHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	user := validateRequest(r)

	if user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	packageName := fmt.Sprintf("%s/%s", ps.ByName("owner"), ps.ByName("repo"))

	isDev := false
	if strings.HasSuffix(packageName, "~dev") {
		isDev = true
		packageName = packageName[:len(packageName)-4]
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	singleResponse := make(map[string]interface{})
	singleResponse["minified"] = "composer/2.0"
	versions := make([]map[string]interface{}, 0)

	err := db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("packages")).Cursor()

		prefix := []byte(packageName + "|")
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			if isDev && !strings.HasPrefix(string(k), packageName+"|dev-") {
				continue
			}

			if !isDev && strings.HasPrefix(string(k), packageName+"|dev-") {
				continue
			}

			composerJson := map[string]interface{}{}

			if err := json.Unmarshal(v, &composerJson); err != nil {
				return err
			}

			versions = append(versions, composerJson)
		}

		return nil
	})

	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	singleResponse["packages"] = map[string]interface{}{packageName: versions}

	err = json.NewEncoder(w).Encode(singleResponse)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func addOrUpdateVersion(tx *bolt.Tx, bytes []byte, version, downloadLink string) error {
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
