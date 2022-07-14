# Composer Registry

This tool fetches all branches, tags of configured repositories and provides a Packagist API.

## Why?

As many `vcs` repositories are so slow in doing `composer update`

## Installation

- Build the go project
- Create a `config.yml`

```yaml
users:
  - token: test

providers:
  - name: internal-gitlab
    type: gitlab
    domain: gitlab.com
    token: GITLAB_TOKEN
    webhook_secret: test
    projects:
      - name: shopware/6/product/platform
```

Add following to your `composer.json`

```json
"repositories": [
    {
        "type": "composer",
        "url": "<url-of-this-service-hosted>"
    }
],
"config": {
    "gitlab-domains": ["<GITLAB-DOMAIN>"]
}
```

The packagist links direct to your Gitlab Instance, so you need to authenticate your local composer with an Gitlab API Token

```shell
> composer config gitlab-token.<GITLAB-DOMAIN> TOKEN
```

To get updates add a Push event webhook to `/webhook` and configure the secret correctly in `GITLAB_WEBHOOK_SECRET` var.
