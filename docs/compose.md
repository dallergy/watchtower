# Compose and environment variables

Watchtower is a single container. Notifications (Gotify, and optionally other services) run inside it. Do not add an Apprise service.

## Quick start

1. Copy the example env file:
   ```bash
   cp .env.example .env
   ```
2. Edit `.env`. For Gotify set `WATCHTOWER_NOTIFICATIONS=gotify`, `WATCHTOWER_NOTIFICATION_GOTIFY_URL`, and `WATCHTOWER_NOTIFICATION_GOTIFY_TOKEN`.
3. Start Watchtower:
   ```bash
   docker compose up -d
   ```

The repository `docker-compose.yml` is deploy-ready: one Watchtower service only.

## Compose file

```yaml
services:
  watchtower:
    image: shounak6942/watchtower:latest
    container_name: watchtower
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    environment:
      TZ: UTC
      WATCHTOWER_CLEANUP: "true"
      WATCHTOWER_POLL_INTERVAL: "86400"
      WATCHTOWER_NOTIFICATION_REPORT: "true"
      WATCHTOWER_NO_STARTUP_MESSAGE: "true"
      WATCHTOWER_NOTIFICATIONS: gotify
      WATCHTOWER_NOTIFICATIONS_LEVEL: info
      WATCHTOWER_NOTIFICATION_GOTIFY_URL: https://gotify.example.com/
      WATCHTOWER_NOTIFICATION_GOTIFY_TOKEN: your.gotify.application.token
```

Build a local image instead of pulling:

```yaml
    build:
      context: .
      dockerfile: dockerfiles/Dockerfile.dev-self-contained
    image: watchtower:local
```

## Environment variable guide

Every CLI flag has a matching `WATCHTOWER_*` (or Docker) environment variable. Boolean flags accept `true` / `false`.

### Required / core

| Variable | Default | Purpose |
| --- | --- | --- |
| `TZ` | `UTC` | Time zone for logs and cron schedules |
| `WATCHTOWER_CLEANUP` | `false` | Delete the previous image after a successful update |
| `WATCHTOWER_POLL_INTERVAL` | `86400` | Seconds between registry checks |
| `WATCHTOWER_SCHEDULE` | unset | Cron expression; when set, overrides the poll interval |
| `WATCHTOWER_TIMEOUT` | `10s` | How long to wait for a container to stop |
| `WATCHTOWER_RUN_ONCE` | `false` | Check once and exit |

The Docker socket must be mounted at `/var/run/docker.sock` (read-only is enough).

### Notifications

Gotify is built into Watchtower and talks to your Gotify server directly.

| Variable | Default | Purpose |
| --- | --- | --- |
| `WATCHTOWER_NOTIFICATIONS` | unset | Set to `gotify` (add `email`, `slack`, or `msteams` if needed) |
| `WATCHTOWER_NOTIFICATION_GOTIFY_URL` | unset | Gotify base URL, e.g. `https://gotify.example.com/` |
| `WATCHTOWER_NOTIFICATION_GOTIFY_TOKEN` | unset | Gotify application token |
| `WATCHTOWER_NOTIFICATION_GOTIFY_TLS_SKIP_VERIFY` | `false` | Skip TLS verify (testing only) |
| `WATCHTOWER_NOTIFICATIONS_LEVEL` | `info` | Minimum log level that is forwarded to notifications |
| `WATCHTOWER_NOTIFICATION_REPORT` | `false` | Send a session summary instead of raw log lines |
| `WATCHTOWER_NOTIFICATION_TEMPLATE` | built-in | Custom Go template for the message body |
| `WATCHTOWER_NOTIFICATIONS_HOSTNAME` | container hostname | Hostname shown in the title |
| `WATCHTOWER_NOTIFICATIONS_DELAY` | `0` | Seconds to wait before sending |
| `WATCHTOWER_NOTIFICATION_TITLE_TAG` | unset | Prefix in the notification title |
| `WATCHTOWER_NOTIFICATION_SKIP_TITLE` | `false` | Do not send a title |
| `WATCHTOWER_NO_STARTUP_MESSAGE` | `false` | Do not notify when Watchtower starts |
| `WATCHTOWER_NOTIFICATION_URL` | unset | `gotifys://host/token` or other bundled service URLs |
| `WATCHTOWER_NOTIFICATION_APPRISE_CONFIG` | unset | Optional Apprise config file path inside this container |

Legacy email/slack/msteams flags still work. Prefer Gotify's dedicated variables above, or `WATCHTOWER_NOTIFICATION_URL` for other Apprise services.

### Container selection

| Variable | Purpose |
| --- | --- |
| `WATCHTOWER_LABEL_ENABLE` | Only update containers with `com.centurylinklabs.watchtower.enable=true` |
| `WATCHTOWER_DISABLE_CONTAINERS` | Comma/space-separated container names to skip |
| `WATCHTOWER_MONITOR_ONLY` | Detect updates but do not recreate containers |
| `WATCHTOWER_NO_PULL` | Do not pull; only use images already on the host |
| `WATCHTOWER_NO_RESTART` | Pull (and optionally clean up) without recreating |
| `WATCHTOWER_INCLUDE_STOPPED` | Also consider created/exited containers |
| `WATCHTOWER_REVIVE_STOPPED` | Start stopped containers after they are updated |
| `WATCHTOWER_INCLUDE_RESTARTING` | Include containers in `restarting` state |
| `WATCHTOWER_SCOPE` | Only update containers with a matching scope label |
| `WATCHTOWER_ROLLING_RESTART` | Restart one container at a time |
| `WATCHTOWER_LIFECYCLE_HOOKS` | Run pre-/post-update commands from labels |
| `WATCHTOWER_REMOVE_VOLUMES` | Remove anonymous volumes when recreating |
| `WATCHTOWER_LABEL_TAKE_PRECEDENCE` | Container labels override CLI/env |

You can also pass container names as the service `command` so only those names are watched.

### HTTP API and metrics

| Variable | Purpose |
| --- | --- |
| `WATCHTOWER_HTTP_API_UPDATE` | Only update when `POST /v1/update` is called |
| `WATCHTOWER_HTTP_API_PERIODIC_POLLS` | Keep the schedule as well as the HTTP trigger |
| `WATCHTOWER_HTTP_API_METRICS` | Expose Prometheus metrics at `/v1/metrics` |
| `WATCHTOWER_HTTP_API_TOKEN` | Bearer token required by the HTTP API |

Map host port `8080` if you enable the API. There is no bundled Grafana or Prometheus.

### Logging and Docker

| Variable | Purpose |
| --- | --- |
| `WATCHTOWER_DEBUG` / `WATCHTOWER_TRACE` | Verbose logs (`trace` can leak credentials) |
| `WATCHTOWER_LOG_LEVEL` | `panic`, `fatal`, `error`, `warn`, `info`, `debug`, `trace` |
| `WATCHTOWER_LOG_FORMAT` | `Auto`, `LogFmt`, `Pretty`, `JSON` |
| `NO_COLOR` | Disable ANSI colors |
| `DOCKER_HOST` | Docker daemon URL (default `unix:///var/run/docker.sock`) |
| `DOCKER_API_VERSION` | Docker API version |
| `DOCKER_TLS_VERIFY` | Verify TLS when using a TCP Docker host |
| `REPO_USER` / `REPO_PASS` | Private registry credentials (or mount `config.json`) |

A full flag list lives in [Arguments](arguments.md). Notification URL formats live in [Notifications](notifications.md).
