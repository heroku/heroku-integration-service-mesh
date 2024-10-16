# Heroku Integration Service Mesh

The Heroku Integration Service Mesh is a software layer that front's Heroku Integration managed apps providing 
connected client communication authentication. It accompanies the 
[Heroku Integration add-on](https://devcenter.heroku.com/articles/heroku-integration).

## Overview

The Heroku Integration Service Mesh is installed by
[Heroku Buildpack for Heroku Integration Service Mesh](https://github.com/heroku/heroku-buildpack-heroku-integration-service-mesh),
is started in the app's [Procfile](https://devcenter.heroku.com/articles/procfile) web process type, and listens on external
`$PORT`. Service mesh Managed apps listen on internal `$APP_PORT`.

![Heroku Integration Service Mesh diagram](/docs/heroku-integration-service-mesh-diagram.png)

The Heroku Integration Service Mesh provides the following capabilities:

- **App startup and health:** With a provided app startup command, the service mesh ensures that the app is running 
and healthy.
- **Authentication**: Only connected clients can only invoke a target service mesh managed app. Each app is 
connected to a Salesforce or Data Cloud org by a developer or admin via the Heroku Integration CLI plugin. Each 
request travels with a secure identity that is then validated and authenticated by the Heroku Integration Service 
Mesh and Heroku Integration add-on.
- **Metrics:** The service mesh captures request metrics.


## Developing
See [DEVELOPING.md](docs/DEVELOPING.md).