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
	"path"
	"sort"
	"strings"
)

var config *Config
var db *bolt.DB

func main() {
	router := httprouter.New()
	router.GET("/packages.json", packagesJsonHandler)
	router.GET("/p/:owner/:repo/versions.json", singlePackageHandler)
	router.POST("/webhook/:name", webhookHandler)
	router.GET("/custom/:owner/:repo/:version/file.zip", handleCustomDownload)

	var err error
	config, err = LoadConfig()
	if err != nil {
		log.Fatalln(err)
	}

	db, err = bolt.Open(path.Join(config.StoragePath, "packages.db"), 0666, nil)
	if err != nil {
		log.Fatalln(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("packages"))
		return err
	})

	defer db.Close()

	registerProviders(config, router)

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
		c := tx.Bucket([]byte("packages")).Cursor()

		prefix := []byte("packages--")
		for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			k = bytes.TrimPrefix(k, prefix)

			packageNameSplit := strings.Split(string(k), "|")
			availablePackagesIndexed[packageNameSplit[0]] = true
		}

		return nil
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

		prefix := []byte("packages--" + packageName + "|")
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			k = bytes.TrimPrefix(k, []byte("packages--"))
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

	singleResponse["packages"] = map[string]interface{}{packageName: optimizeComposerVersions(versions)}

	err = json.NewEncoder(w).Encode(singleResponse)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func handleCustomDownload(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	user := validateRequest(r)

	if user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	packageName := fmt.Sprintf("%s/%s", ps.ByName("owner"), ps.ByName("repo"))

	zipFile := getZipPath(packageName, ps.ByName("version"))

	http.ServeFile(w, r, zipFile)
}
