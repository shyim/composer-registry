# Gitlab Registry composer

This tool fetches all branches, tags of configured repositories and provides a Packagist API.

## Why?

A many `vcs` repositories are so slow in doing `composer update`

## Installation

- Build the go project
- Set following environment variables

Environment variables

- `GITLAB_PROJECTS` - List of projects to fetch. Example: `1,2,3,4` (project ids)
- `GITLAB_TOKEN` - Gitlab private token to talk with API
- `GITLAB_DOMAIN` - Gitlab domain to talk with API
- `GITLAB_WEBHOOK_SECRET` - Webhook Secret

Add following to your `composer.json`

```json
"repositories": [
    {
        "type": "composer",
        "url": "<url-of-this-service-hosted>"
    }
],
"config": {
    "gitlab-domains": ["gitlab.shopware.com"]
}
```

The packagist links direct to your Gitlab Instance, so you need to authenticate your local composer with an Gitlab API Token

```shell
> composer config gitlab-token.<GITLAB-DOMAIN> TOKEN
```

To get updates add a Push event webhook to `/webhook` and configure the secret correctly in `GITLAB_WEBHOOK_SECRET` var.
