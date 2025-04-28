## Developing Heroku AppLink Service Mesh

### Prerequisites

1. Install `go`.
2. Install `go` dependencies.

### Test Locally

1. Build `heroku-applink-service-mesh`.

```shell
$ make build
▶ formatting …
▶ vetting …
▶ testing …
ok      command-line-arguments  0.009s
▶ done
```

2. Run Heroku AppLink add-on service locally or run fake service locally that mocks Heroku AppLink's authentication APIs.

```shell
$ npm start

> fake-heroku-applink@1.0.0 start
> fastify start -l info app.js

{"level":30,"time":1727726903891,"pid":414136,"hostname":"cwall-wsl10","msg":"Server listening at http://127.0.0.1:3000"}
```

To change log level, set `GO_LOG` env var on stat command line:

```shell
$ GO_LOG=debug...
```

3. Start local app with `heroku-applink-service-mesh`.

```shell
$ pwd
/home/cwall/git/heroku-sf-AppLink-nodejs

$ HEROKU_APPLINK_TOKEN=TOKEN HEROKU_APPLINK_API_URL=http://localhost:3000 APP_PORT=8080 /home/cwall/git/heroku-applink-service-mesh/bin/heroku-applink-service-mesh npm start
time=2024-09-30T15:41:57.421-06:00 level=INFO msg=environment app=local source=heroku-applink-service-mesh go_version:=go1.22.1 os=linux arch=amd64 http_port=8070 version=v0.12.1 environment=local app_port=8080
time=2024-09-30T15:41:57.421-06:00 level=INFO msg="router running" app=local source=heroku-applink-service-mesh port=8070

> heroku-salesforce-api-fastify-app@1.0.0 start
> fastify start -a 0.0.0.0 -p $APP_PORT -l debug src/app.js

{"level":30,"time":1727732518332,"pid":509954,"hostname":"cwall-wsl10","msg":"Server listening at http://0.0.0.0:8080"}
```

4. Invoke app API proxying through `heroku-applink-service-mesh`.

```shell
$ curl -v -X POST -H "Content-Type: application/json" -H "x-request-id: MYREQUESTID" -H "x-signature: SIGNATURE" 'http://0.0.0.0:8070/handleDataCloudDataChangeEvent?orgId=ORGID&apiName=DAT' -d @sample-data-action-target-webhook-request.json
[Service mesh and app log]
time=2024-09-30T15:58:07.632-06:00 level=INFO msg="Processing request to /handleDataCloudDataChangeEvent..." app=local source=heroku-applink-service-mesh request-id=MYREQUESTID
time=2024-09-30T15:58:07.632-06:00 level=INFO msg="Validating request..." app=local source=heroku-applink-service-mesh request-id=MYREQUESTID
time=2024-09-30T15:58:07.632-06:00 level=INFO msg="Found Data Action Target request" app=local source=heroku-applink-service-mesh request-id=MYREQUESTID
time=2024-09-30T15:58:07.632-06:00 level=INFO msg="Authenticating Data Action Target 'DAT' request from org ORGID with payload length 4856..." app=local source=heroku-applink-service-mesh request-id=MYREQUESTID
time=2024-09-30T15:58:07.678-06:00 level=INFO msg="Authenticated request!" app=local source=heroku-applink-service-mesh request-id=MYREQUESTID
time=2024-09-30T15:58:07.678-06:00 level=INFO msg="Forwarding request..." request-id=MYREQUESTID
{"level":30,"time":1727733487690,"pid":518808,"hostname":"cwall-wsl10","reqId":"req-1","req":{"method":"POST","url":"/handleDataCloudDataChangeEvent?orgId=ORGID&apiName=DAT","hostname":"127.0.0.1:8080","remoteAddress":"127.0.0.1","remotePort":38774},"msg":"incoming request"}
2024/09/30 15:58:07 [MYREQUESTID] "POST http://0.0.0.0:8070/handleDataCloudDataChangeEvent?orgId=ORGID&apiName=DAT HTTP/1.1" from 127.0.0.1:60580 - 201 0B in 69.478122ms
{"level":30,"time":1727733487697,"pid":518808,"hostname":"cwall-wsl10","reqId":"req-1","msg":"POST /dataCloudDataChangeEvent: 1 events for schemas Test_DataCloud_SDK_Action"}
{"level":30,"time":1727733487697,"pid":518808,"hostname":"cwall-wsl10","reqId":"req-1","msg":"Got action 'Test_DataCloud_SDK_Action', event type 'CDCEvent' triggered by INSERT on object 'Heroku_Click_Events__dlm' published on 2024-09-04T21:47:01.798Z"}
{"level":30,"time":1727733487704,"pid":518808,"hostname":"cwall-wsl10","reqId":"req-1","res":{"statusCode":201},"responseTime":12.573991000652313,"msg":"request completed"}
```

Invalid request

```shell
$ curl -v -X POST -H "Content-Type: application/json" -H "x-request-id: MYREQUESTID" -H "x-request-context: {}" -H "x-client-context: {}" 'http://0.0.0.0:8070/accounts' -d '{"events": [{"id":"idhere"}] }'
time=2024-09-30T16:01:54.865-06:00 level=INFO msg="Processing request to /accounts..." app=local source=heroku-applink-service-mesh request-id=00Dxx0000000000EAA-7c566091-7af3-4e87-8865-4e014444c298-2024-09-03T20:56:27.608444Z
time=2024-09-30T16:01:54.865-06:00 level=INFO msg="Validating request..." app=local source=heroku-applink-service-mesh request-id=00Dxx0000000000EAA-7c566091-7af3-4e87-8865-4e014444c298-2024-09-03T20:56:27.608444Z
time=2024-09-30T16:01:54.865-06:00 level=ERROR msg="Invalid request!" app=local source=heroku-applink-service-mesh request-id=00Dxx0000000000EAA-7c566091-7af3-4e87-8865-4e014444c298-2024-09-03T20:56:27.608444Z
time=2024-09-30T16:01:54.865-06:00 level=ERROR msg="400 Invalid request" app=local source=heroku-applink-service-mesh request-id=00Dxx0000000000EAA-7c566091-7af3-4e87-8865-4e014444c298-2024-09-03T20:56:27.608444Z
```

### Release

1. Update `conf/version.go` bumping up appropriate major, minor, or patch version (vX.Y.Z) via PR.
1. Once your PR is merged, switch to the main branch `git checkout main`.
1. Get the commit SHA of your merged PR on main.
1. Run `make release VERSION=vX.Y.Z SHA=1234ab` This runs `scripts/release.sh` which contains some sanity checks.
1. Confirm that you want to proceed after sanity checks pass. The script creates and pushes a tag to the repo which triggers the Github Action in `.github/workflows/release.yml`.

```shell
% make release VERSION=v0.0.3 SHA=1234abc
▶ create release tag and push to github to trigger release
All Pre-checks passed. Release version v0.0.3 at commit 1234abc? (y/N) y
...
To https://github.com/heroku/heroku-applink-service-mesh.git
 * [new tag]         v0.0.3 -> v0.0.3
Release v0.0.3 created on commit 1234abc and pushed successfully. Check https://github.com/heroku/heroku-applink-service-mesh/actions/workflows/release.yml for GH Action status.
```

5. Run buildpack locally to validate download and install. See [Heroku Buildpack for Heroku AppLink Service Mesh - Run Locally](https://github.com/heroku/heroku-buildpack-heroku-applink-service-mesh?tab=readme-ov-file#run-locally).
