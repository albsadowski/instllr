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
* creates nginx virtual host config file `/etc/nginx/sites-enabled/<my-host.domain.com>.conf`.

Next, the service can be run and/or enabled: `systemctl enable --now <my-host.domain.com>`.

In case you don't have the SSL certificates already in place, run: `certbot certonly --standalone`.

Restart nginx: `systemctl restart nginx`.

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
