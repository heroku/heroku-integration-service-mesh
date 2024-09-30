# Heroku Integration Service Mesh

The service mesh is a proxy that front's the customers' app providing authentication and authorization. It
accompanies the Integration Add-on intercepting incoming requests for validation and capabilities.

![service mesh proxy diagram](/docs/diagram.png)


## Overview

The service mesh currently provides the following capabilities:

- **Start a customer's app:** With a customer provided command, the service mesh ensures that the app is running and healthy.
- **Authentication**: Known clients can only invoke a target app or resource. Each app is connected to orgs. Connected orgs are able to invoke the heroku integration app. Each request travels with a C2C JWT to ensure that the org is registered with the Heroku Integration app.
- **Authorization**: Provides pre-configured, scoped tokens for app/resources to external services.

The service mesh component is deployed alongside the app provided by [Heroku Buildpack for Heroku Integration Service Mesh](https://github.com/heroku/heroku-buildpack-heroku-integration-service-mesh).

## Development
See [DEVELOPING.md](docs/DEVELOPING.md).