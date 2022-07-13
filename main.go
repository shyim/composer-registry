package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/xanzy/go-gitlab"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var packages = make(map[string]map[string]interface{})
var gitlabProjects = strings.Split(os.Getenv("GITLAB_PROJECTS"), ",")
var gitlabDomain = os.Getenv("GITLAB_DOMAIN")
var gitlabWebookSecret = os.Getenv("GITLAB_WEBHOOK_SECRET")
var git *gitlab.Client

func main() {
	var err error
	git, err = gitlab.NewClient(os.Getenv("GITLAB_TOKEN"), gitlab.WithBaseURL(fmt.Sprintf("https://%s/api/v4", gitlabDomain)))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/packages.json", packagesJsonHandler)
	mux.HandleFunc("/webhook", webhookHandler)

	go updateAll()

	http.ListenAndServe(":8080", mux)
}

func webhookHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(writer, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if request.Header.Get("X-Gitlab-Token") != gitlabWebookSecret {
		http.Error(writer, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	var event gitlab.PushEvent

	if err := json.NewDecoder(request.Body).Decode(&event); err != nil {
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	version := ""

	if strings.HasPrefix("refs/tags/", event.Ref) {
		version = strings.ToLower(strings.TrimPrefix(event.Ref, "refs/tags/"))
	} else {
		version = strings.ToLower(fmt.Sprintf("dev-%s", strings.TrimPrefix(event.Ref, "refs/heads/")))
	}

	if err := addOrUpdate(strconv.FormatInt(int64(event.ProjectID), 10), version, event.CheckoutSHA); err != nil {
		log.Printf(err.Error())
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func packagesJsonHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	err := json.NewEncoder(w).Encode(map[string]interface{}{"packages": packages})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func updateAll() {
	for _, project := range gitlabProjects {
		if err := updateAllTags(project); err != nil {
			log.Println(err)
		}
		if err := updateAllBranches(project); err != nil {
			log.Println(err)
		}
	}
}

func updateAllBranches(gitlabId string) error {
	page := 1

	for {
		branches, _, err := git.Branches.ListBranches(gitlabId, &gitlab.ListBranchesOptions{
			ListOptions: gitlab.ListOptions{PerPage: 100, Page: page},
		})

		if err != nil {
			return err
		}

		for _, branch := range branches {
			log.Printf("Fetching infos for branch: %s\n", branch.Name)
			if err := addOrUpdate(gitlabId, strings.ToLower(fmt.Sprintf("dev-%s", branch.Name)), branch.Commit.ShortID); err != nil {
				return err
			}
		}

		if len(branches) != 100 {
			break
		}
		page = page + 1
	}

	return nil
}

func updateAllTags(gitlabId string) error {
	page := 1

	for {
		tags, _, err := git.Tags.ListTags(gitlabId, &gitlab.ListTagsOptions{
			ListOptions: gitlab.ListOptions{PerPage: 100, Page: page},
		})

		if err != nil {
			return err
		}

		for _, tag := range tags {
			log.Printf("Fetching infos for tag: %s\n", tag.Name)
			if err := addOrUpdate(gitlabId, strings.ToLower(fmt.Sprintf("dev-%s", tag.Name)), tag.Commit.ShortID); err != nil {
				return err
			}
		}

		if len(tags) != 100 {
			break
		}
		page = page + 1
	}

	return nil
}

func addOrUpdate(pid, version, sha string) error {
	file, _, err := git.RepositoryFiles.GetFile(pid, "composer.json", &gitlab.GetFileOptions{Ref: &sha})

	if err != nil {
		return err
	}

	bytes, _ := base64.StdEncoding.DecodeString(file.Content)

	composerJson := map[string]interface{}{}

	if err := json.Unmarshal(bytes, &composerJson); err != nil {
		return err
	}

	packageName := composerJson["name"].(string)

	if _, ok := packages[packageName]; !ok {
		packages[packageName] = make(map[string]interface{})
	}

	composerJson["dist"] = map[string]string{
		"url":  fmt.Sprintf("https://%s/api/v4/projects/%s/repository/archive.zip?sha=%s", gitlabDomain, pid, sha),
		"type": "zip",
	}

	composerJson["version"] = version

	packages[packageName][version] = composerJson

	return nil
}
