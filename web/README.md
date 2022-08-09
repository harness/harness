# Sample Module UI

## Prerequisites

```
yarn setup-github-registry
```

## Local development

Change current directory to policy-mgmt project folder and run API server:

```
APP_ENABLE_UI=false APP_ENABLE_STANDALONE=true APP_TOKEN_JWT_SECRET=1234 APP_INTERNAL_TOKEN_JWT_SECRET=5678 APP_HTTP_BIND=localhost:3001 go run main.go server
```

### Run the UI as a standalone app

```
yarn
yarn dev
```

Wait until Webpack build is done, then access http://localhost:3002/#/signin.

Note that you can point standalone UI app to a non-local backend service by creating a `.env` (under `web` or project folder) with content looks like:

```
TARGET_LOCALHOST=false
BASE_URL=https://qa.harness.io/gateway
```

### Run the UI as a micro-frontend service

Due to an issue with Webpack (reason still unknown), you can't mount micro-frontend app inside NextGen UI when it's being run under Webpack development mode (aka `yarn dev`). To overcome the issue, run:

```
yarn
yarn micro:watch
```

The micro front-end UI will be served under http://localhost:3000. Run [Core UI](https://github.com/harness/harness-core-ui/) locally and navigate to the app within NextGen UI.

## Build

UI build is integrated a a part of the backend build. See `.drone.yml` and `Taskfile.yml` for more information.
