# Getting started

Choose the source branch for the interface shown in the project README, or the published container for a first look without Git or a build. The published 0.4.2 artifacts and current source have different interfaces. [Requirements and platforms](../README.md#requirements-and-platforms).

## Source workspace

With Git and Docker Compose v2 installed:

```sh
git clone --branch feat/scalable-wos-operation https://github.com/A1b3rt0M3rcad0/wos.git
cd wos
cp .env.example .env
docker compose up --build -d
```

Open `http://127.0.0.1:8080/app/`. Use the `WOS_BOOTSTRAP_TOKEN` value from your `.env` as **Access credential**, then select **Sign in**. The unchanged example opens the **Main** Workspace. Begin with **+** beside **Outcomes**; the [workspace guide](workspace-ux.md) explains planning, Tasks, Evidence, and explicit conclusions.

> [!WARNING]
> `.env.example` contains a public development credential. The Compose port is bound to localhost. Before sharing an instance, configure unique private credentials, permissions, and HTTPS using [shared-service operations](operations.md#shared-service). Keep credentials out of logs, URLs, screenshots, and Git.

## Published container

This pinned historical release uses the old `NAMESPACE` environment variables below. They are retained only for this released image; current source uses Tenant contracts and requires the [breaking cutover](scaling/tenants.md#upgrade-an-existing-installation).

The public Linux amd64 image `ghcr.io/a1b3rt0m3rcad0/wos:0.4.2` runs without cloning this repository. This command starts the released service with a persistent SQLite volume and a localhost-only port:

```sh
docker run --detach --name wos-local \
  --publish 127.0.0.1:8080:8080 \
  --volume wos_local_data:/data \
  --env WOS_LISTEN=0.0.0.0:8080 \
  --env WOS_AUTH_MODE=api_token \
  --env WOS_LOCAL_PRINCIPAL_ID=operator \
  --env WOS_BOOTSTRAP_TOKEN=wos-local-development-bootstrap-token-change-for-shared-instances \
  --env WOS_BOOTSTRAP_NAMESPACE_ID=0199ff31-0000-7000-8000-000000000001 \
  --env WOS_BOOTSTRAP_NAMESPACE_NAME=Main \
  ghcr.io/a1b3rt0m3rcad0/wos:0.4.2
```

Open `http://127.0.0.1:8080/app/` and sign in with the public local-example credential supplied in the command:

```text
wos-local-development-bootstrap-token-change-for-shared-instances
```

> [!WARNING]
> This is a public demo credential, suitable only for this localhost example. Use a private credential and protected configuration for a shared service. The image does not print your credential in its logs.

The release manifest is anonymously readable on GHCR. The verified 0.4.2 image digest is `sha256:6a626ffc55fcb05f36c424a778fb8029598dd3594b386d4e441395150e80e1f4`; the release source is `b572269085494ddd4d6370716ff1cb360b46065f`. The image has the earlier released UI, rather than the editorial branch screenshots. See [release formats and verification](releases.md).

Use `docker logs --tail=100 wos-local`, `docker stop wos-local`, and `docker start wos-local` for this container. `docker rm wos-local` removes a stopped container while retaining the named `wos_local_data` volume. A container-name conflict means that name already exists; inspect it or choose another name instead of removing an unknown container.

## Build from source

Install stable Go 1.27.2, pinned in [`.go-version`](../.go-version). The [Go language baseline](../go.mod) is 1.27. No Go toolchain is needed to run a released native archive or container.

From a checkout of this branch:

```sh
go version
go build -trimpath -o bin/wos ./packages/wos-api/cmd/wos
./bin/wos config validate
./bin/wos server
```

The default source server is trusted-local mode. It binds loopback and uses SQLite. Open `http://127.0.0.1:8080/app/`, expand **Local development server**, enter Tenant `0199d000-0000-7000-8000-000000000001`, and choose **Open local workspace**. Local identity is a development composition, not multiuser authentication. Configure a [shared service](operations.md#shared-service) for multiple participants.

The binary embeds the web workspace and five skills. It also provides API-only client commands; those commands do not open the server database. The [isolated unsigned example](../examples/quickstart/README.md) uses this binary to run the complete first-contact workflow.

## Sign-in and persisted data

Bootstrap initializes credentials when the database is first created. Replacing `WOS_BOOTSTRAP_TOKEN` in `.env` later does not change an existing credential, restore revoked access, or bypass authentication. Use the credential from the original initialization. If you still have an administrator credential, follow the credential operations in the [shared-service guide](operations.md#shared-service). Loss of all administrator credentials needs the controlled offline recovery described there.

An unsigned local demonstration does not configure independent-review enforcement. Signed execution requires the appropriate [profiles](signed-workspace-profiles.md) and [protocol cutover](signed-protocol-cutover.md). Independent review requires eligible distinct Principals and its applicable policy; unsigned deployments can also enforce it with `WOS_INDEPENDENT_REVIEWER=true`. [Agent operations](agents.md) explains the separate steps.

## Service lifecycle

For the source Compose setup:

```sh
docker compose logs -f wos
docker compose stop
docker compose start
docker compose down
```

SQLite data remains in the `wos_data` volume when containers stop or are removed by `down`. Starting the same deployment with that volume resumes the same state. A shared URL opens context but grants no access.

> [!WARNING]
> `docker compose down -v` removes volumes and their data. Back up state before any destructive maintenance. See [backup and restore](operations.md#backup-restore-and-upgrade).

## Setup troubleshooting

| Problem                                       | Action                                                                                                                                                                              |
| --------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Port 8080 is busy                             | For Compose, use `WOS_HTTP_PORT=8081 docker compose up --build -d`; open `http://127.0.0.1:8081/app/`. For `docker run`, change the host mapping to `127.0.0.1:8081:8080`.          |
| Docker permission denied / cannot connect     | Check `docker info`, start your Docker service or desktop application, and follow Docker's supported user-access setup for your OS.                                                 |
| Connection refused while the service starts   | Check `docker compose ps` and `docker compose logs --tail=100 wos`; wait for the service to become ready.                                                                           |
| Credential rejected                           | Check the whole credential and the first-initialization configuration. See [sign-in and persisted data](#sign-in-and-persisted-data).                                               |
| Image cannot run on this architecture         | The validated image is Linux amd64. macOS/ARM native CLI distributions and container emulation are not qualified. Use a supported server and connect through your browser.          |
| Source build cannot verify TLS behind a proxy | Configure the host's proxy and CA trust. The [proxy build guide](operations.md#builds-behind-an-https-proxy) supports a build-only trust bundle without disabling TLS verification. |

For PostgreSQL, protected issuer material, upgrades, and monitoring, use [operations](operations.md) and the [shared deployment reference](../deploy/shared/README.md).
