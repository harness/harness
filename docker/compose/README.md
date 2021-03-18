# Local development

This directory contains Docker compose files used by the core development team for local development and testing purposes only. These are not part of the core distribution, and are not intended for use outside of the core development team. We are not currently accepting changes or additions to these files.

## Running a Drone deployment locally using Github

At the end of this guide you will have a drone server and a drone runner that is hooked up to your Github account. This will allow you to trigger builds on your Github repositories.

### (prerequisite) Setup a Github oauth application

Create an oauth application here <https://github.com/settings/developers>

The most important entry is setting the `Authorization callback URL` you can set this to `http://localhost:8080/login`

You will also need to create a client secret for the application.

Now you have the `DRONE_GITHUB_CLIENT_ID` and `DRONE_GITHUB_CLIENT_SECRET`

### (prerequisite) Setup Ngrok

Ngrok allows us to send the webhooks from Github to our local Drone setup.

Follow the guide here <https://dashboard.ngrok.com/get-started/setup>

### Running Drone

+ Move into the `drone/docker/compose/drone-github` folder.

+ Run Ngrok against port `8080` it will run in the foreground.

``` bash
./ngrok http 8080
```

Take note of the forwarding hostname this is your `DRONE_SERVER_PROXY_HOST` EG

``` bash
Forwarding    http://c834c33asdde.ngrok.io -> http://localhost:8080
```

+ You will want to edit the Docker compose file `docker-compose.yml` updating in the following entries.

``` bash
DRONE_SERVER_PROXY_HOST=${DRONE_SERVER_PROXY_HOST} # taken from Ngrok
DRONE_GITHUB_CLIENT_ID=${DRONE_GITHUB_CLIENT_ID}   # taken from your Github oauth application
DRONE_GITHUB_CLIENT_ID=${DRONE_GITHUB_CLIENT_ID}   # taken from your Github oauth application
```

NB for `DRONE_SERVER_PROXY_HOST` do not include http/https.

+ Run docker compose

``` bash
docker-compose up
```

Now you can go access the Drone ui at <http://localhost:8080>
