package main

import (
	"context"
	"fmt"
	"github.com/google/go-github/v45/github"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/oauth2"
	"net/http"
	"strings"
)

type GithubProvider struct {
	provider ConfigProvider
	client   *github.Client
}

func NewGithubProvider(provider ConfigProvider) GithubProvider {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: provider.Token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return GithubProvider{provider: provider, client: github.NewClient(tc)}
}

func (g GithubProvider) GetConfig() ConfigProvider {
	return g.provider
}

func (g GithubProvider) UpdateAll() error {
	for _, project := range g.provider.Projects {
		nameSplit := strings.Split(project.Name, "/")

		if err := g.updateAllTags(context.Background(), nameSplit[0], nameSplit[1]); err != nil {
			log.Errorf("cannot update all tags %s", err.Error())
		}

		if err := g.updateAllBranches(context.Background(), nameSplit[0], nameSplit[1]); err != nil {
			log.Errorf("cannot update all tags %s", err.Error())
		}
	}

	return nil
}

func (g GithubProvider) Webhook(request *http.Request) error {
	payload, err := github.ValidatePayload(request, []byte(g.provider.WebhookSecret))

	if err != nil {
		return err
	}

	event, err := github.ParseWebHook(github.WebHookType(request), payload)

	if err != nil {
		return err
	}

	switch event := event.(type) {
	case *github.PushEvent:
		version := ""

		if strings.HasPrefix("refs/tags/", event.GetRef()) {
			version = strings.ToLower(strings.TrimPrefix(event.GetRef(), "refs/tags/"))
		} else {
			version = strings.ToLower(fmt.Sprintf("dev-%s", strings.TrimPrefix(event.GetRef(), "refs/heads/")))
		}

		return db.Update(func(tx *bolt.Tx) error {
			return g.addOrUpdate(context.Background(), tx, event.GetRepo().GetOwner().GetName(), event.GetRepo().GetName(), version, event.GetAfter())
		})
	default:
		return fmt.Errorf("invalid webhook type")
	}
}

func (g GithubProvider) updateAllTags(ctx context.Context, owner string, repo string) error {
	return db.Batch(func(tx *bolt.Tx) error {
		page := 1

		for {
			tags, _, err := g.client.Repositories.ListTags(ctx, owner, repo, &github.ListOptions{PerPage: 100, Page: page})

			if err != nil {
				return err
			}

			for _, tag := range tags {
				if err := g.addOrUpdate(ctx, tx, owner, repo, tag.GetName(), tag.GetCommit().GetSHA()); err != nil {
					log.Errorf("cannot update tag %s of %s/%s\n", tag.GetName(), owner, repo)
				}
			}

			if len(tags) != 100 {
				break
			}

			page++
		}

		return nil
	})
}

func (g GithubProvider) updateAllBranches(ctx context.Context, owner string, repo string) error {
	return db.Batch(func(tx *bolt.Tx) error {
		page := 1

		for {
			branches, _, err := g.client.Repositories.ListBranches(ctx, owner, repo, &github.BranchListOptions{ListOptions: github.ListOptions{PerPage: 100, Page: page}})

			if err != nil {
				return err
			}

			for _, tag := range branches {
				if err := g.addOrUpdate(ctx, tx, owner, repo, tag.GetName(), tag.GetCommit().GetSHA()); err != nil {
					log.Errorf("cannot update branch %s of %s/%s\n", tag.GetName(), owner, repo)
				}
			}

			if len(branches) != 100 {
				break
			}

			page++
		}

		return nil
	})
}

func (g GithubProvider) addOrUpdate(ctx context.Context, tx *bolt.Tx, owner, repo, version, sha string) error {
	log.Infof("updating info of %s/%s for version %s\n", owner, repo, version)

	file, _, _, err := g.client.Repositories.GetContents(ctx, owner, repo, "composer.json", &github.RepositoryContentGetOptions{Ref: sha})

	if err != nil {
		return err
	}

	content, err := file.GetContent()

	if err != nil {
		return err
	}

	return addOrUpdate(tx, []byte(content), version, fmt.Sprintf("https://api.github.com/repos/%s/%s/zipball/%s", owner, repo, sha))
}
