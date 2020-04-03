# Draft Content API

[![Circle CI](https://circleci.com/gh/Financial-Times/draft-content-api/tree/master.png?style=shield)](https://circleci.com/gh/Financial-Times/draft-content-api/tree/master)[![Go Report Card](https://goreportcard.com/badge/github.com/Financial-Times/draft-content-api)](https://goreportcard.com/report/github.com/Financial-Times/draft-content-api) [![Coverage Status](https://coveralls.io/repos/github/Financial-Times/draft-content-api/badge.svg)](https://coveralls.io/github/Financial-Times/draft-content-api)

## Introduction

Draft content API is a microservice that provides access to draft content stored in PAC.
At the moment the service is a simple proxy to UPP Content API.

## Installation

Download the source code, dependencies and test dependencies:

        curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
        mkdir $GOPATH/src/github.com/Financial-Times/draft-content-api
        cd $GOPATH/src/github.com/Financial-Times
        git clone https://github.com/Financial-Times/draft-content-api.git
        cd public-things-api && dep ensure -vendor-only
        go build .

## Running locally

1. Run the tests and install the binary:

        go test ./...
        go install

2. Run the binary (using the `help` flag to see the available optional arguments):

        $GOPATH/bin/draft-content-api [--help]

Options:

        --app-system-code="draft-content-api"    System Code of the application ($APP_SYSTEM_CODE)
        --app-name="draft-content-api"           Application name ($APP_NAME)
        --app-timeout="8s"                       Endpoints Timeout Duration ($APP_TIMEOUT)
        --port="8080"                            Port to listen on ($APP_PORT)
        --content-rw-endpoint="..."              Endpoint for draft content RW ($DRAFT_CONTENT_RW_ENDPOINT)
        --mam-endpoint="..."                     Endpoint for draft content mapper ($DRAFT_CONTENT_MAPPER_ENDPOINT)
        --content-endpoint="..."                 Endpoint to get content from CAPI ($CONTENT_ENDPOINT)
        --content-api-key="..."                  API key to access CAPI ($CAPI_APIKEY)


3. Test:

    1. Either using curl:

            curl http://localhost:8080/draft/content/b7b871f6-8a89-11e4-8e24-00144feabdc0 | json_pp

    1. Or using [httpie](https://github.com/jkbrzt/httpie):

            http GET http://localhost:8080/draft/content/b7b871f6-8a89-11e4-8e24-00144feabdc0

## Build and deployment

* Built by Docker Hub on merge to master: [coco/draft-content-api](https://hub.docker.com/r/coco/draft-content-api/)
* CI provided by CircleCI: [draft-content-api](https://circleci.com/gh/Financial-Times/draft-content-api)

## Service endpoints

Note that the read endpoint is `/drafts/content/{uuid}` whereas the write endpoint is `/drafts/nativecontent/{uuid}`.

### GET

Using curl:

    curl http://localhost:8080/drafts/content/b7b871f6-8a89-11e4-8e24-00144feabdc0 | json_pp`

Or using [httpie](https://github.com/jkbrzt/httpie):

    http GET http://localhost:8080/drafts/content/b7b871f6-8a89-11e4-8e24-00144feabdc0

At the moment this endpoint is a proxy to the content available in UPP,
so it returns a payload consistent to the Content API in UPP.

### PUT

Using curl:

    curl -X PUT http://localhost:8080/drafts/nativecontent/b7b871f6-8a89-11e4-8e24-00144feabdc0 --data-binary "@/path/to/file.json"

This returns a 200 status with no body.

## Healthchecks
Admin endpoints are:

`/__gtg`

`/__health`

`/__build-info`

The `/__health` and `/__gtg` check the availability of:
* the generic R/W service (where draft content is stored in native format)
* the draft content mapper service (where draft content is mapped from native to UPP format)
* the UPP Content API (where published content is stored)


### Logging

* The application uses [logrus](https://github.com/sirupsen/logrus); the log file is initialised in [main.go](main.go).
* Logging requires an `env` app parameter, for all environments other than `local` logs are written to file.
* When running locally, logs are written to console. If you want to log locally to file, you need to pass in an env parameter that is != `local`.
* NOTE: `/__build-info` and `/__gtg` endpoints are not logged as they are called every second from varnish/vulcand and this information is not needed in logs/splunk.

## Change/Rotate sealed secrets

Please reffer to documentation in [pac-global-sealed-secrets-eks](https://github.com/Financial-Times/pac-global-sealed-secrets-eks/blob/master/README.md). Here are explained details how to create new, change existing sealed secrets.
