# instllr

Systemd service installer.

## Motivation

Super naive service installer that helps me deploy my side projects.

## How does it work?

1. A service you'd like to deploy, is packaged (tar xz) and available as a GitHub Release.
2. The root of the package contains `instllr.json` - a descriptor file containing instructions on how the service should be installed and run.
3. On a target host, run the `instllr`:
```sh
GH_TOKEN=<gh-token> \
NODE_PATH=<path-to-nodejs-bin> \
instllr --host <my-host.domain.com> \
        --port <local-port>
        --app-env <some env; e.g., config file path> \
        install <owner>/<repo>:<tag>
```
4. `instllr` fetches the GitHub release asset and:
* creates a user and a group `<my-host.domain.com>`,
* untars the asset into `/home/<my-host.domain.com>/...`,
* creates systemd service file `/etc/systemd/system/<my-host.domain.com>.service`,
* creates nginx virtual host config file `/etc/nginx/sites-enabled/<my-host.domain.com>.conf`,
* enables the service `systemctl enable --now <my-host.domain.com>`,
* re-starts nginx `systemctl restart nginx`.

An non-standard simplification is hostname == app name.

## Pre-requisites

1. Host running linux with systemd.
2. DNS A record pointing to your host.
3. Certbot standalone certificate available under `/etc/letsencrypt/live/$HOST` (generate one with `certbot certonly --standalone`).

## instllr.json

Example:

```json
{
    "require": [
        {
            "app": "node",
            "version": ["node", "--version"],
            "minVersion": "18.15.0"
        },
    ],
    "env": {
        "require": ["NODE_ENV"]
    },
    "install": ["npm", "install", "--omit", "dev"],
    "run": ["node", "main.js"]
}
```

## Cron

If you drop `instllr` command with a `latest` tag into your crontab, you have a poor man's continuous delivery, e.g.:

```
* * * * * NODE_PATH=/opt/node-v18.15.0/bin instllr --app-env NODE_ENV=production --host <app-host> --port <local-port> install <gh-user>/<gh-repo> 2>&1 | logger -t CRON
```
