# update-detector-linebot

The LINE bot to notify web pages update. (for GAE)

## Usage

### 1. Add the `app.yaml`

``` yaml:app.yaml
application: yourapplicationname
version: 1
runtime: go
api_version: go1
handlers:
- url: /.*
  script: _go_app
env_variables:
  PATH_SUFFIX: '-your-path-suffix'  # It is for to keep the endpoints undiscovered by others.
  CHANNEL_SECRET: 'Your LINE channel secret'
  CHANNEL_TOKEN: 'Your LINE channel token'
```

### 2. Add the `cron.yaml`

``` yaml:cron.yaml
cron:
- description: "check update"
  url: /cron-your-path-suffix   # '/cron' + $PATH_SUFFIX
  schedule: every 5 minutes
```

### 3. Deploy to GAE

``` bash
$ goapp deploy
```

## Endpoints

- /callback _-your-path-suffix_ ... for the LINE Mesanger API Webhook
- /cron _-your-path-suffix_ ... for checking updates
