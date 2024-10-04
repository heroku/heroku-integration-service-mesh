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
2. Run Heroku Integration add-on service locally or run fake service locally that mocks Heroku Integration's authentication APIs.
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

$ GO_LOG=debug HEROKU_INTEGRATION_TOKEN=TOKEN HEROKU_INTEGRATION_API_URL=http://localhost:3000 APP_PORT=8080 /home/cwall/git/heroku-integration-service-mesh/bin/heroku-integration-service-mesh npm start
time=2024-09-30T15:41:57.421-06:00 level=INFO msg=environment app=local source=heroku-integration-service-mesh go_version:=go1.22.1 os=linux arch=amd64 http_port=8070 version=v0.12.1 environment=local app_port=8080
time=2024-09-30T15:41:57.421-06:00 level=INFO msg="router running" app=local source=heroku-integration-service-mesh port=8070

> heroku-salesforce-api-fastify-app@1.0.0 start
> fastify start -a 0.0.0.0 -p $APP_PORT -l debug src/app.js

{"level":30,"time":1727732518332,"pid":509954,"hostname":"cwall-wsl10","msg":"Server listening at http://0.0.0.0:8080"}
```
3. Invoke app API proxying through `heroku-integration-service-mesh`.
```shell
$ curl -v -X POST -H "Content-Type: application/json" -H "x-request-id: MYREQUESTID" -H "x-signature: SIGNATURE" 'http://0.0.0.0:8070/handleDataCloudDataChangeEvent?orgId=ORGID&apiName=DAT' -d @sample-data-action-target-webhook-request.json
[Service mesh and app log]
time=2024-09-30T15:58:07.632-06:00 level=INFO msg="Processing request to /handleDataCloudDataChangeEvent..." app=local source=heroku-integration-service-mesh request-id=MYREQUESTID
time=2024-09-30T15:58:07.632-06:00 level=INFO msg="Validating request..." app=local source=heroku-integration-service-mesh request-id=MYREQUESTID
time=2024-09-30T15:58:07.632-06:00 level=DEBUG msg="Headers: {\"Accept\":[\"*/*\"],\"Content-Length\":[\"4856\"],\"Content-Type\":[\"application/json\"],\"User-Agent\":[\"curl/7.81.0\"],\"X-Request-Id\":[\"MYREQUESTID\"],\"X-Signature\":[\"SIGNATURE\"]}" app=local source=heroku-integration-service-mesh request-id=MYREQUESTID
time=2024-09-30T15:58:07.632-06:00 level=INFO msg="Found Data Action Target request" app=local source=heroku-integration-service-mesh request-id=MYREQUESTID
time=2024-09-30T15:58:07.632-06:00 level=INFO msg="Authenticating Data Action Target 'DAT' request from org ORGID with payload length 4856..." app=local source=heroku-integration-service-mesh request-id=MYREQUESTID
time=2024-09-30T15:58:07.633-06:00 level=DEBUG msg="!! REMOVEME !! Calling Heroku Integration API http://localhost:3000/connections/datacloud/authenticate [TOKEN] with body {\"api_name\":\"DAT\",\"org_id\":\"ORGID\",\"signature\":\"SIGNATURE\",\"payload\":\"{...}\"}" app=local source=heroku-integration-service-mesh request-id=MYREQUESTID
time=2024-09-30T15:58:07.678-06:00 level=INFO msg="Response for Data Action Target authentication request (/connections/datacloud/authenticate): statusCode 200, body ''" app=local source=heroku-integration-service-mesh request-id=MYREQUESTID
time=2024-09-30T15:58:07.678-06:00 level=DEBUG msg="Data Action Target authentication took 45.579358ms" app=local source=heroku-integration-service-mesh request-id=MYREQUESTID
time=2024-09-30T15:58:07.678-06:00 level=INFO msg="Authenticated request!" app=local source=heroku-integration-service-mesh request-id=MYREQUESTID
time=2024-09-30T15:58:07.678-06:00 level=INFO msg="Forwarding request to http://127.0.0.1:8080/handleDataCloudDataChangeEvent?orgId=ORGID&apiName=DAT" app=local source=heroku-integration-service-mesh request-id=MYREQUESTID
{"level":30,"time":1727733487690,"pid":518808,"hostname":"cwall-wsl10","reqId":"req-1","req":{"method":"POST","url":"/handleDataCloudDataChangeEvent?orgId=ORGID&apiName=DAT","hostname":"127.0.0.1:8080","remoteAddress":"127.0.0.1","remotePort":38774},"msg":"incoming request"}
time=2024-09-30T15:58:07.701-06:00 level=DEBUG msg="Heroku Integration Service Mesh took 69.348357ms" app=local source=heroku-integration-service-mesh request-id=MYREQUESTID
2024/09/30 15:58:07 [MYREQUESTID] "POST http://0.0.0.0:8070/handleDataCloudDataChangeEvent?orgId=ORGID&apiName=DAT HTTP/1.1" from 127.0.0.1:60580 - 201 0B in 69.478122ms
{"level":30,"time":1727733487697,"pid":518808,"hostname":"cwall-wsl10","reqId":"req-1","msg":"POST /dataCloudDataChangeEvent: 1 events for schemas Test_DataCloud_SDK_Action"}
{"level":30,"time":1727733487697,"pid":518808,"hostname":"cwall-wsl10","reqId":"req-1","msg":"Got action 'Test_DataCloud_SDK_Action', event type 'CDCEvent' triggered by INSERT on object 'Heroku_Click_Events__dlm' published on 2024-09-04T21:47:01.798Z"}
{"level":30,"time":1727733487704,"pid":518808,"hostname":"cwall-wsl10","reqId":"req-1","res":{"statusCode":201},"responseTime":12.573991000652313,"msg":"request completed"}

$ curl -v -X POST -H "Content-Type: application/json" -H "x-request-id: MYREQUESTID" -H "x-request-context: {}" -H "x-client-context: {}" 'http://0.0.0.0:8070/accounts' -d '{"events": [{"id":"idhere"}] }'
[Service mesh and app log]
time=2024-09-30T16:01:54.865-06:00 level=INFO msg="Processing request to /accounts..." app=local source=heroku-integration-service-mesh request-id=00Dxx0000000000EAA-7c566091-7af3-4e87-8865-4e014444c298-2024-09-03T20:56:27.608444Z
time=2024-09-30T16:01:54.865-06:00 level=INFO msg="Validating request..." app=local source=heroku-integration-service-mesh request-id=00Dxx0000000000EAA-7c566091-7af3-4e87-8865-4e014444c298-2024-09-03T20:56:27.608444Z
time=2024-09-30T16:01:54.865-06:00 level=DEBUG msg="Headers: {\"Accept\":[\"*/*\"],\"Content-Type\":[\"application/json\"],\"User-Agent\":[\"curl/7.81.0\"],\"X-Client-Context\":[\"ewogICJyZXF1ZXN0SWQiOiAiMDBEeHgwMDAwMDAwMDAwRUFBLTdjNTY2MDkxLTdhZjMtNGU4Ny04ODY1LTRlMDE0NDQ0YzI5OC0yMDI0LTA5LTAzVDIwOjU2OjI3LjYwODQ0NFoiLAogICJhY2Nlc3NUb2tlbiI6ICJBQ0NFU1NfVE9LRU4iLAogICJhcGlWZXJzaW9uIjogIjYyLjAiLAogICJuYW1lc3BhY2UiOiAiIiwKICAib3JnSWQiOiAiMDBEeHgwMDAwMDAwMDAwRUFBIiwKICAib3JnRG9tYWluVXJsIjogIk9SR19ET01BSU4iLAogICJ1c2VyQ29udGV4dCI6IHsKICAgICJ1c2VySWQiOiAiMDA1eHgwMDAwMDFYN3E5QUFDIiwKICAgICJ1c2VybmFtZSI6ICJhZG1pbkBteWNvbXBhbnkuY29tIgogIH0KfQo=\"],\"X-Request-Id\":[\"00Dxx0000000000EAA-7c566091-7af3-4e87-8865-4e014444c298-2024-09-03T20:56:27.608444Z\"]}" app=local source=heroku-integration-service-mesh request-id=00Dxx0000000000EAA-7c566091-7af3-4e87-8865-4e014444c298-2024-09-03T20:56:27.608444Z
time=2024-09-30T16:01:54.865-06:00 level=ERROR msg="Invalid request! Invalid Salesforce header(s): Invalid x-request-context header" app=local source=heroku-integration-service-mesh request-id=00Dxx0000000000EAA-7c566091-7af3-4e87-8865-4e014444c298-2024-09-03T20:56:27.608444Z
time=2024-09-30T16:01:54.865-06:00 level=ERROR msg="400 Invalid request" app=local source=heroku-integration-service-mesh request-id=00Dxx0000000000EAA-7c566091-7af3-4e87-8865-4e014444c298-2024-09-03T20:56:27.608444Z
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
