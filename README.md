# Composer Registry

This Composer registry fetches multiple sources and serves them as Composer packages.  

## Features

- Gitlab Support
- GitHub Support
- Mirror Shopware Composer Repository
- Adding custom packages using ZIP files

## Installation

- Download the latest version from the released files
- Create a `config.json` or a `config.yml` like your preferences.

## Configuration

The base config file looks like this:

```json
{
  "$schema": "https://raw.githubusercontent.com/shyim/composer-registry/main/config-schema.json",
  "base_url": "http://localhost:8080"
}
```

The `base_url` is the URL where the instance is available. Currently it is not possible to change this afterwards. (This will be used only for mirroring/custom packages).

### Gitlab

```javascript
{
    "$schema": "https://raw.githubusercontent.com/shyim/composer-registry/main/config-schema.json",
    "base_url": "http://localhost:8080",
    "providers": [
        {
            "name": "my-gitlab-instance", // provider name.
            "type": "gitlab",
            "domain": "gitlab.com", // gitlab.com or your instance domain
            "token": "my-gitlab-token", // access token with read_api
            "webhook_secret": "my-gitlab-webhook-secret", // webhook secret Webhook address is /webhook/<provider-name> 
            "fetch_all_on_start": true, // Fetches all packages on start, Optional
            "projects": [
                {
                    "name": "my-gitlab-group/repo" // project name to consider
                }
            ],
            "cron_schedule": "*/5 * * * *" // Cron schedule for refetching anything. Optional if you don't want to have webhooks
        }
    ]
}
```

The registry serves only the package information, the zip will be directly downloaded from your Gitlab instance. To do this you need to configure composer too.

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

```shell
> composer config gitlab-token.<GITLAB-DOMAIN> TOKEN
```

### Github

```javascript
{
    "$schema": "https://raw.githubusercontent.com/shyim/composer-registry/main/config-schema.json",
    "base_url": "http://localhost:8080",
    "providers": [
        {
            "name": "my-github-instance", // provider name.
            "type": "github",
            "token": "my-github-token", // personal access token
            "webhook_secret": "my-github-webhook-secret", // webhook secret Webhook address is /webhook/<provider-name> 
            "fetch_all_on_start": true, // Fetches all packages on start, Optional
            "projects": [
                {
                    "name": "my-github-group/repo" // project name to consider
                }
            ],
            "cron_schedule": "*/5 * * * *" // Cron schedule for refetching anything. Optional if you don't want to have webhooks
        }
    ]
}
```

The registry serves only the package information, the zip will be directly downloaded from your GitHub instance. To do this you need to configure composer too.

Add following to your `composer.json`

```shell
> composer config github-oauth.github.com token
```

### Mirroring Shopware Composer

```javascript
{
    "$schema": "https://raw.githubusercontent.com/shyim/composer-registry/main/config-schema.json",
    "base_url": "http://localhost:8080",
    "providers": [
        {
            "name": "shopware", // provider name.
            "type": "shopware",
            "fetch_all_on_start": true, // Fetches all packages on start, Optional
            "projects": [
                {
                    "name": "The Composer Token" // project name to consider
                }
            ],
            "cron_schedule": "*/5 * * * *" // Cron schedule for refetching anything. Optional if you don't want to have webhooks
        }
    ]
}
```

The registry will download all packages from the Shopware Composer repository and serve them.

### Adding custom zips as package

```javascript
{
    "$schema": "https://raw.githubusercontent.com/shyim/composer-registry/main/config-schema.json",
    "base_url": "http://localhost:8080",
    "providers": [
        {
            "name": "custom", // provider name.
            "type": "custom",
            "webhook_secret": "my-api-secret"
        }
    ]
}
```

This will enable two API endpoints to add/update or delete packages:

Create/update package:


```http request
POST http://localhost:8080/custom/package/create
Authorization: bearer <webhook-secret>
```

The request body is the ZIP file.


Delete package:

```http request
DELETE http://localhost:8080/custom/package/<name>/<version>
Authorization: bearer <webhook-secret>
```

# Authentication

By default, is the authentication disabled. To enabled add users to your configuration.

```json
{
    "$schema": "https://raw.githubusercontent.com/shyim/composer-registry/main/config-schema.json",
    "base_url": "http://localhost:8080",
    "users": [
        {
            "token": "TOKEN"
        }
    ]
}
```

The token can be configured like so

```shell
> composer config bearer.<instance-domain> <your-token>
```

The users will have always access to all packages currently.

