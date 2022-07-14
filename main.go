package main

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func main() {
	router := httprouter.New()
	router.GET("/packages.json", packagesJsonHandler)
	router.POST("/webhook/:name", webhookHandler)

	config, err := LoadConfig()
	if err != nil {
		log.Fatalln(err)
	}

	registerProviders(config)

	go updateAll()

	log.Fatal(http.ListenAndServe(":8080", router))
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

	err := json.NewEncoder(w).Encode(map[string]interface{}{"packages": packages})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func addOrUpdate(bytes []byte, version, downloadLink string) error {
	composerJson := map[string]interface{}{}

	if err := json.Unmarshal(bytes, &composerJson); err != nil {
		return err
	}

	packageName := composerJson["name"].(string)

	if _, ok := packages[packageName]; !ok {
		packages[packageName] = make(map[string]interface{})
	}

	composerJson["dist"] = map[string]string{
		"url":  downloadLink,
		"type": "zip",
	}

	composerJson["version"] = version

	packages[packageName][version] = composerJson

	return nil
}

func updateAll() {
	for name, provider := range providers {
		log.Infof("Updating all packages of %s", name)
		if err := provider.UpdateAll(); err != nil {
			log.Infof("Error updating all packages of %s: %s", name, err)
		}
	}
}
