package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"
	bolt "go.etcd.io/bbolt"
)

type GitlabProvider struct {
	Provider ConfigProvider
	git      *gitlab.Client
}

func NewGitlabProvider(provider ConfigProvider) GitlabProvider {
	var err error
	git, err := gitlab.NewClient(provider.Token, gitlab.WithBaseURL(fmt.Sprintf("https://%s/api/v4", provider.Domain)))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	return GitlabProvider{Provider: provider, git: git}
}

func (g GitlabProvider) GetConfig() ConfigProvider {
	return g.Provider
}

func (g GitlabProvider) UpdateAll() error {
	for _, project := range g.Provider.Projects {
		if err := g.updateAllTags(project.Name); err != nil {
			return err
		}
		if err := g.updateAllBranches(project.Name); err != nil {
			return err
		}
	}

	return nil
}

func (g GitlabProvider) Webhook(request *http.Request) error {
	if request.Header.Get("X-Gitlab-Token") != g.Provider.WebhookSecret {
		return fmt.Errorf("forbidden")
	}

	var event gitlab.PushEvent

	if err := json.NewDecoder(request.Body).Decode(&event); err != nil {
		return err
	}

	version := ""
	trimmedVersion := ""

	if strings.HasPrefix("refs/tags/", event.Ref) {
		version = strings.ToLower(strings.TrimPrefix(event.Ref, "refs/tags/"))
		trimmedVersion = version
	} else {
		trimmedVersion = strings.ToLower(strings.TrimPrefix(event.Ref, "refs/heads/"))
		version = "dev-" + trimmedVersion
	}

	saveTag := g.generateSaveTag(event.ProjectID, trimmedVersion)

	return db.Update(func(tx *bolt.Tx) error {
		if event.After == "0000000000000000000000000000000000000000" {
			return deleteVersion(tx, saveTag)
		}
		return g.addOrUpdate(tx, strconv.FormatInt(int64(event.ProjectID), 10), version, event.CheckoutSHA, saveTag)
	})
}

func (g GitlabProvider) generateSaveTag(projectID int, trimmedVersion string) string {
	return fmt.Sprintf("%d-%s", projectID, trimmedVersion)
}

func (GitlabProvider) RegisterCustomHTTPHandlers(router *httprouter.Router) {

}

func (g GitlabProvider) updateAllBranches(gitlabId string) error {
	project, _, err := g.git.Projects.GetProject(gitlabId, &gitlab.GetProjectOptions{})

	if err != nil {
		return err
	}

	return db.Batch(func(tx *bolt.Tx) error {
		page := 1

		for {
			branches, _, err := g.git.Branches.ListBranches(gitlabId, &gitlab.ListBranchesOptions{
				ListOptions: gitlab.ListOptions{PerPage: 100, Page: page},
			})

			if err != nil {
				return err
			}

			for _, branch := range branches {
				log.Printf("Fetching infos for repo %s and branch: %s\n", gitlabId, branch.Name)

				saveTag := g.generateSaveTag(project.ID, branch.Name)

				if err := g.addOrUpdate(tx, strconv.FormatInt(int64(project.ID), 10), strings.ToLower(fmt.Sprintf("dev-%s", branch.Name)), branch.Commit.ShortID, saveTag); err != nil {
					return err
				}
			}

			if len(branches) != 100 {
				break
			}
			page = page + 1
		}

		return nil
	})
}

func (g GitlabProvider) updateAllTags(gitlabId string) error {
	project, _, err := g.git.Projects.GetProject(gitlabId, &gitlab.GetProjectOptions{})

	if err != nil {
		return err
	}

	return db.Batch(func(tx *bolt.Tx) error {
		page := 1

		for {
			tags, _, err := g.git.Tags.ListTags(gitlabId, &gitlab.ListTagsOptions{
				ListOptions: gitlab.ListOptions{PerPage: 100, Page: page},
			})

			if err != nil {
				return err
			}

			for _, tag := range tags {
				log.Printf("Fetching infos for repo %s and tag: %s\n", gitlabId, tag.Name)
				saveTag := g.generateSaveTag(project.ID, tag.Name)
				if err := g.addOrUpdate(tx, strconv.FormatInt(int64(project.ID), 10), strings.ToLower(tag.Name), tag.Commit.ShortID, saveTag); err != nil {
					return err
				}
			}

			if len(tags) != 100 {
				break
			}
			page = page + 1
		}

		return nil
	})
}

func (g GitlabProvider) addOrUpdate(tx *bolt.Tx, pid, version, sha, saveTag string) error {
	file, _, err := g.git.RepositoryFiles.GetFile(pid, "composer.json", &gitlab.GetFileOptions{Ref: &sha})

	if err != nil {
		return err
	}

	bytes, _ := base64.StdEncoding.DecodeString(file.Content)

	return addOrUpdateVersion(tx, bytes, version, fmt.Sprintf("https://%s/api/v4/projects/%s/repository/archive.zip?sha=%s", g.Provider.Domain, pid, sha), saveTag)
}
