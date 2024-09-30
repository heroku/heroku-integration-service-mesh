## Developing Heroku Integration Service Mesh

### Prerequisites
1. Install `go`.
2. Install `go` dependencies.

### Test Locally
1. Build `heroku-integration-service-mesh`.
```shell
$ make build
▶ formatting …
▶ vetting …
▶ building bin/heroku-integration-service-mesh …
▶ done
``` 
2. Run Heroku Integration add-on service locally or build and run fake service locally.
```shell
$ npm start

> fake-heroku-integration@1.0.0 start
> fastify start -l info app.js

{"level":30,"time":1727726903891,"pid":414136,"hostname":"cwall-wsl10","msg":"Server listening at http://127.0.0.1:3000"}
```
2. Start local app with `heroku-integration-service-mesh`.
```shell
$ pwd
/home/cwall/git/heroku-sf-integration-nodejs

$ GO_LOG=debug HEROKU_INTEGRATION_INVOCATIONS_TOKEN=TOKEN HEROKU_INTEGRATION_API_URL=http://localhost:3000 APP_PORT=8080 /home/cwall/git/heroku-integration-service-mesh/bin/heroku-integration-service-mesh npm start
time=2024-09-30T15:41:57.421-06:00 level=INFO msg=environment app=local source=heroku-integration-service-mesh go_version:=go1.22.1 os=linux arch=amd64 http_port=8070 version=v0.12.1 environment=local app_port=8080
time=2024-09-30T15:41:57.421-06:00 level=INFO msg="router running" app=local source=heroku-integration-service-mesh port=8070

> heroku-salesforce-api-fastify-app@1.0.0 start
> fastify start -a 0.0.0.0 -p $APP_PORT -l debug src/app.js

{"level":30,"time":1727732518332,"pid":509954,"hostname":"cwall-wsl10","msg":"Server listening at http://0.0.0.0:8080"}
```

### Release

1. Update `version.go` bumping up appropriate major, minor, or patch version.
2. On [heroku/heroku-integration-service repo](https://github.com/heroku/heroku-integration-service-mesh/releases), follow [Create a release](https://docs.github.com/en/repositories/releasing-projects-on-github/managing-releases-in-a-repository#creating-a-release) instructions to create a release.
3. Run `make release` to build and generate `heroku-integration-service-mesh.tar.gz` tar file.
```shell
$ make release
▶ formatting …
▶ vetting …
▶ done
▶ tar heroku-integration-service-mesh ...
heroku-integration-service-mesh

$ ls -l heroku-integration-service-mesh.tar.gz 
-rw-rw-r-- 1 cwall cwall 6.7M Sep 30 15:25 heroku-integration-service-mesh.tar.gz
```
4. Upload `heroku-integration-service-mesh.tar.gz` as release artifact.
5. Run buildpack locally to validate download and install.  See [Heroku Buildpack for Heroku Integration Service Mesh - Run Locally](https://github.com/heroku/heroku-buildpack-heroku-integration-service-mesh?tab=readme-ov-file#run-locally).
