# Heroku Integration Service Mesh

The Heroku Integration Service Mesh is a software layer that front's Heroku AppLink managed apps (formerly known as Integration apps) providing
connected client communication authentication. It accompanies the
[Heroku Integration add-on](https://devcenter.heroku.com/articles/heroku-integration).

## Overview
### Architecture
- The Service Mesh runs as a proxy layer in front of the actual app
- The Service Mesh listens on the external `$PORT`
- Service Mesh managed apps listen on the internal `$APP_PORT`
- The Heroku Integration Service Mesh is installed by
[Heroku Buildpack for Heroku Integration Service Mesh](https://github.com/heroku/heroku-buildpack-heroku-integration-service-mesh)
- The Heroku Integration Service Mesh is started in the app's [Procfile](https://devcenter.heroku.com/articles/procfile) web process type

### Request Flow
- Incoming requests hit the service mesh
- The service mesh validates and authenticates the request
- If authorized, the request is then forwarded to the customer's app running on `$APP_PORT`
- The customer's app processes the request and sends the response back through the service mesh

### Authentication Endpoints
- After validation the Service Mesh handles authentication and forwards successfuly authenticated requests to the relevant API endpoint:
    1. Salesforce:
        - HerokuIntegrationSalesforceAuthPath (/invocations/authentication)
    2. Data Action Target:
        - HerokuIntegrationDataActionTargetAuthPath (/data_action_targets/authenticate)
- Once authenticated the request is forwarded to the target app

### Architecture Diagram
![Heroku Integration Service Mesh diagram](/docs/heroku-integration-service-mesh-diagram.png)

## Service Mesh Capabilities:

- **App startup and health:**
    - With a provided app startup command, the service mesh ensures that the app is running
and healthy.
- **Authentication**
    - Only connected clients can only invoke a target service mesh managed app. Each app is
connected to a Salesforce or Data Cloud org by a developer or admin via the Heroku Integration CLI plugin. Each
request travels with a secure identity that is then validated and authenticated by the Heroku Integration Service
Mesh and Heroku Integration add-on.
- **Metrics:**
    - The service mesh captures request metrics.


## Developing
See [DEVELOPING.md](docs/DEVELOPING.md).
