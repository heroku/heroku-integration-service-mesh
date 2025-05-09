# Heroku AppLink Service Mesh

The Heroku AppLink Service Mesh is a software layer that front's Heroku AppLink managed apps providing
[Heroku AppLink add-on](https://devcenter.heroku.com/articles/heroku-applink).

## Overview

The Heroku AppLink Service Mesh is installed by
[Heroku Buildpack for Heroku AppLink Service Mesh](https://github.com/heroku/heroku-buildpack-heroku-applink-service-mesh),
is started in the app's [Procfile](https://devcenter.heroku.com/articles/procfile) web process type, and listens on external
`$PORT`. Service mesh Managed apps listen on internal `$APP_PORT`.

![Heroku AppLink Service Mesh diagram](/docs/heroku-applink-service-mesh-diagram.png)

The Heroku AppLink Service Mesh provides the following capabilities:

- **App startup and health:** With a provided app startup command, the service mesh ensures that the app is running
  and healthy.
- **Authentication**: Only connected clients can only invoke a target service mesh managed app. Each app is
  connected to a Salesforce or Data Cloud org by a developer or admin via the Heroku AppLink CLI plugin. Each
  request travels with a secure identity that is then validated and authenticated by the Heroku AppLink Service
  Mesh and Heroku AppLink add-on.
- **Metrics:** The service mesh captures request metrics.

## Developing

See [DEVELOPING.md](docs/DEVELOPING.md).
