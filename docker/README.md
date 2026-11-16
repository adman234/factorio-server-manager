# Factorio Server Manager Docker Image

## Prerequisites
You need to have [Docker](https://www.digitalocean.com/community/tutorials/how-to-install-and-use-docker-on-ubuntu-20-04)
and [Docker Compose](https://docs.docker.com/compose/install/) installed.

## Getting started?

Copy `docker-compose.yaml` and `.env` files from this repository to somewhere on your server.

Edit values in the `.env` file:
* `RCON_PASS` (default empty string): Password for Factorio RCON (FSM uses it to communicate with the Factorio server). \
  If left empty, a random password will be generated and saved on the first start of the server. You can see the password in `fsm-data/conf.json` file.
* `DOMAIN_NAME` (must be set manually): The domain name where your FSM UI will be available. Must be set,
  so [Let's Encrypt](https://letsencrypt.org/) service can issue a valid HTTPS certificate for this domain.
* `EMAIL_ADDRESS` (must be set manually): Your email address. Used only by Let's Encrypt service.

Alternatively you can ignore `.env` file and edit this values directly in `environment` section of `docker-compose.yaml`.
But remember that if `.env` file is present, values set there take precedence over values set in `docker-compose.yaml`.

Now you can start the container by running:

```
docker compose up -d
```

### Environment variables

| Variable | Default | Description |
| --- | --- | --- |
| `FACTORIO_VERSION` | `stable` | `stable`, `latest` (experimental) or an exact version such as `2.0.72`. |
| `FACTORIO_ALLOW_MAJOR_UPGRADE` | `false` | Allow `stable`/`latest` to move to a new major release (e.g. 1.1 → 2.0). |
| `RCON_PASS` | empty | RCON password. Applied on every start; generated and saved to `fsm-data/conf.json` when empty. |
| `FSM_SECURE_COOKIES` | `false` | Mark the login cookie `Secure`. Set to `true` only when FSM is served over HTTPS (the Traefik compose file does this). |
| `FSM_AUTOSTART` | `false` | Start the Factorio server with the latest save when the container starts. |
| `PUID` / `PGID` | unset (root) | Run FSM and Factorio as this user and group. Volume contents are `chown`ed to them on start. |

`docker stop` saves the map before Factorio exits. Keep `stop_grace_period` (60s in the compose files) long enough for a large map to save.

### Simple configuration without HTTPS

If you don't care about HTTPS and want to run just the Factorio Server Manager, or want to run it on a local machine you can use `docker-compose.simple.yaml`.

Ignore `DOMAIN_NAME` and `EMAIL_ADDREESS` variables in `.env` file and run
```
docker compose -f docker-compose.simple.yaml up -d
```

### Factorio version

By default the container installs the newest stable version of Factorio. To use a specific version, change
`FACTORIO_VERSION` in the compose file. `latest` installs the newest experimental version.

Factorio is only downloaded when the installed version differs from the requested one, and a failed download
keeps the installed version. While following `stable` or `latest`, the container will not move to a new major
release on its own; it logs a warning instead, so you can back up your saves first.

## Accessing the application

Go to the domain specified in your `.env` file in your web browser. If running on localhost access the application at http://localhost

### First start

When container starts it begins to download Factorio headless server archive, and only after that Factorio Server Manager server starts.
So when docker-compose writes
```
Creating factorio-server-manager ... done
```
you have to wait several seconds before FSM UI becomes available.

It may take some time for Let's Encrypt to issue the certificate, so for the first couple of minutes after starting the container you may see
"Your connection is not private" error when you open your Factorio Server Manager address in your browser. This error should disappear within
a couple of minutes, if configuration parameters are set correctly.

## Updating Credentials, adding and deleting users.

An admin user is created initially using the credentials defined in the factorio-server-manager config file. When you haven't specified one, a random password will be logged in the container output (`docker logs factorio-server-manager`).

Users can be added and deleted on the settings page.

## Updating Factorio

For now, you can't update/downgrade the Factorio version from the UI.

You can however do this using docker images while sustaining your security settings and map/modfiles.

If you want to update Factorio to the latest version, run `docker compose restart` (or
`docker compose -f docker-compose.simple.yaml restart` if you are using the simple configuration).
The running game is saved when the container stops, and the new version is installed when it starts.

## Security

Authentication is supported in the application, but it is recommended to ensure access to the Factorio manager UI is accessible via VPN or internal network.

## Development
For development purposes it also has the ability to create the docker image from local sourcecode. This is done by running `docker/build.sh`, which builds the frontend and backend inside Docker from `docker/Dockerfile`. The created docker image will have the tag `factorio-server-manager:dev`.

### Creating release bundles
A Dockerfile-build file is included for creating the release bundles. Use Docker version 20 in order to use the BUILDKIT environment, some issues have been encountered with Docker version 19.

To create the bundle build the Dockerfile-build file with the following command. The release bundles are output to the ./dist directory.

Run this command from the root factorio-server-manager directory.
```
DOCKER_BUILDKIT=1 docker build --no-cache -f docker/Dockerfile-build -t ofsm-build --target=build -o dist .
```

## For everyone who actually read this thing to the end

And now go and build some nice factories!
