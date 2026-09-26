# Notifications

Watchtower can tell you when it updates containers, fails to update them or runs into problems.
Notification services are configured with [Apprise](https://github.com/caronc/apprise/wiki) style URLs and are
sent directly by Watchtower: there is no Apprise installation, Python runtime or sidecar container involved.

## Quick start

Point `WATCHTOWER_NOTIFICATION_URL` at one or more services, separated by spaces:

=== "compose.yml"

    ```yaml
    services:
      watchtower:
        image: shounak6942/watchtower
        restart: unless-stopped
        volumes:
          - /var/run/docker.sock:/var/run/docker.sock
        environment:
          WATCHTOWER_NOTIFICATION_REPORT: "true"
          WATCHTOWER_NOTIFICATION_URL: >
            gotifys://gotify.example.com/AbCdEfGhIjKlMnO
            ntfys://ntfy.example.com/watchtower
    ```

=== "docker run"

    ```bash
    docker run -d \
      --name watchtower \
      --restart unless-stopped \
      -v /var/run/docker.sock:/var/run/docker.sock \
      -e WATCHTOWER_NOTIFICATION_REPORT=true \
      -e WATCHTOWER_NOTIFICATION_URL="gotifys://gotify.example.com/AbCdEfGhIjKlMnO" \
      shounak6942/watchtower
    ```

With `WATCHTOWER_NOTIFICATION_REPORT` enabled, Watchtower sends one summary per update session, and only when
something was updated or failed. Use `--run-once` with a container name to try your setup without waiting for the
next scheduled check.

## Supported services

| Service         | URL                                                                                                            | Options                                                                                              |
|-----------------|----------------------------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------|
| Gotify          | `gotify://host[:port][/path]/token` (HTTP)<br>`gotifys://host[:port][/path]/token` (HTTPS)                    | `priority`: low, moderate, normal, high, emergency or 0 to 10<br>`format=markdown`                   |
| ntfy            | `ntfy://topic` (on ntfy.sh)<br>`ntfy://[user:pass@]host[:port]/topic` (HTTP)<br>`ntfys://…` (HTTPS)            | `priority`: min, low, default, high, max or 1 to 5<br>`tags`, `click`, `token` (access token)        |
| Slack           | `slack://[botname@]tokenA/tokenB/tokenC[/#channel]`<br>`slack://xoxb-token/#channel`<br>the webhook URL itself | `icon_url`, `icon_emoji`                                                                             |
| Discord         | `discord://[botname@]webhook_id/webhook_token`<br>the webhook URL itself                                       | `avatar_url`, `tts=yes`, `thread` (thread ID)                                                        |
| Telegram        | `tgram://bot_token/chat_id[/chat_id…]`                                                                         | `silent=yes`, `topic` (forum topic ID)                                                               |
| Email           | `mailto://[user:pass@]host[:port]?to=address` (STARTTLS when offered)<br>`mailtos://…` (TLS required)          | `to`, `cc`, `bcc`, `from`, `name`, `smtp` (server), `mode`: starttls, ssl or insecure                |
| Pushover        | `pover://user_key@app_token[/device…]`                                                                         | `priority`: lowest, low, normal, high, emergency or -2 to 2<br>`sound`                               |
| Microsoft Teams | the Power Automate workflow URL<br>`workflows://host[:port]/workflow/signature`                                | Uses an adaptive card. `msteams://team/tokenA/tokenB/tokenC/tokenD` targets retired connector webhooks |
| Webhook         | `json://[user:pass@]host[:port][/path]` (HTTP)<br>`jsons://…` (HTTPS)                                          | `method`: POST, PUT or PATCH<br>`+Header=value` adds a request header                                |

Options are added as query parameters, e.g. `ntfys://ntfy.example.com/updates?priority=high&tags=whale`.
Every web based service also accepts `verify=no` to skip TLS certificate verification, for servers with self-signed
certificates.

!!! tip "Escaping"
    Tokens and passwords that contain characters such as `@`, `/`, `?`, `#` or `%` must be
    [percent-encoded](https://developer.mozilla.org/docs/Glossary/Percent-encoding): a password `p@ss/word` becomes
    `p%40ss%2Fword`.

### Service notes

-   **Gotify**: `gotify://` uses plain HTTP, which suits a Gotify container on the same Docker network
    (`gotify://gotify/token`). Use `gotifys://` for anything reached over HTTPS. The token is sent in the
    `X-Gotify-Key` header rather than the URL.
-   **Email**: the SMTP server is the host of the URL, or the `smtp` option when the host is only used for the default
    sender address (`mailtos://me:app_password@example.com?smtp=smtp.example.com`). Ports default to 25 for
    `mailto://` and 587 for `mailtos://`; port 465 switches to implicit TLS. Credentials are never sent over an
    unencrypted connection unless you set `mode=insecure`. The sender defaults to the user name (or `watchtower@host`)
    and the recipient to the sender.
-   **Slack**: incoming webhooks post to their own channel unless one is given. Bot tokens (`xoxb-…`) need at least one
    channel, written as `#name`, `@user` or `+ID`.
-   **Teams**: Office 365 connectors have been retired by Microsoft. Create a *Post to a channel when a webhook request
    is received* workflow in Teams and use its URL as is.

## Notification URLs

-   `--notification-url` (env. `WATCHTOWER_NOTIFICATION_URL`): The service URLs to notify, separated by spaces. This
    option can also reference a file (for example a Docker secret) that contains one URL per line.
-   `--notification-apprise-config` (env. `WATCHTOWER_NOTIFICATION_APPRISE_CONFIG`): Path *inside the container* to an
    [Apprise configuration file](https://github.com/caronc/apprise/wiki/config). Both the TEXT format (one URL per line)
    and the YAML format (a `urls` list) are read. Per-URL options and tags in YAML files are ignored.

=== "apprise.txt"

    ```text
    # one URL per line, optionally prefixed with tags
    gotifys://gotify.example.com/AbCdEfGhIjKlMnO
    admins = mailtos://watchtower:app_password@smtp.example.com?to=ops@example.com
    ```

=== "apprise.yml"

    ```yaml
    urls:
      - gotifys://gotify.example.com/AbCdEfGhIjKlMnO
      - tgram://123456789:AbCdEfGhIjKlMnOpQrStUvWxYz/-1001234567890
    ```

### Services that are not built in

Apprise supports many more services than Watchtower includes. If you run an
[Apprise API](https://github.com/caronc/apprise-api) server, Watchtower forwards the URLs it cannot handle itself to
it, while the built-in services keep being sent directly:

-   `--notification-apprise-url` (env. `WATCHTOWER_NOTIFICATION_APPRISE_URL`): The Apprise API base URL, for example
    `http://apprise:8000`.
-   `--notification-apprise-key` (env. `WATCHTOWER_NOTIFICATION_APPRISE_KEY`): Optional key of a configuration stored in
    the Apprise API. Notifications are then sent to `/notify/{key}` as well.

Without an Apprise API, a URL for a service that is not built in stops Watchtower at startup with an error that lists
the supported services.

## Settings

-   `--notifications-level` (env. `WATCHTOWER_NOTIFICATIONS_LEVEL`): Controls the log level which is used for the notifications. If omitted, the default log level is `info`. Possible values are: `panic`, `fatal`, `error`, `warn`, `info`, `debug` or `trace`.
-   `--notifications-hostname` (env. `WATCHTOWER_NOTIFICATIONS_HOSTNAME`): Custom hostname specified in subject/title. Useful to override the operating system hostname.
-   `--notifications-delay` (env. `WATCHTOWER_NOTIFICATIONS_DELAY`): Delay before sending notifications expressed in seconds.
-   Watchtower will post a notification every time it is started. This behavior [can be changed](arguments.md#without_sending_a_startup_message) with an argument.
-   `--notification-title-tag` (env. `WATCHTOWER_NOTIFICATION_TITLE_TAG`): Prefix to include in the title. Useful when running multiple watchtowers.
-   `--notification-skip-title` (env. `WATCHTOWER_NOTIFICATION_SKIP_TITLE`): Do not pass the title param to notifications. This will not pass a dynamic title override to notification services. If no title is configured for the service, it will remove the title all together.
-   `--notification-log-stdout` (env. `WATCHTOWER_NOTIFICATION_LOG_STDOUT`): Write rendered notification output to stdout.
-   `--notification-template` (env. `WATCHTOWER_NOTIFICATION_TEMPLATE`): Go [template](https://pkg.go.dev/text/template) used for the message, see [templates](#templates).
-   `--notification-report` (env. `WATCHTOWER_NOTIFICATION_REPORT`): Use the session report as the notification template data.

The URL `logger://` writes notifications to the Watchtower log, which is handy for testing templates.

## Migrating from Shoutrrr

`containrrr/watchtower` used [Shoutrrr](https://containrrr.dev/shoutrrr/) URLs, which look similar to Apprise URLs but
are not the same. The legacy `WATCHTOWER_NOTIFICATIONS` options [below](#legacy_notifications) keep working unchanged,
but `WATCHTOWER_NOTIFICATION_URL` values need to be rewritten:

| Service  | Shoutrrr                                              | This fork (Apprise syntax)                             |
|----------|-------------------------------------------------------|--------------------------------------------------------|
| Gotify   | `gotify://host/token` (HTTPS)                         | `gotifys://host/token` (`gotify://` is plain HTTP)     |
| ntfy     | `ntfy://host/topic` (HTTPS)                           | `ntfys://host/topic`                                   |
| Discord  | `discord://token@webhook_id`                          | `discord://webhook_id/token`                           |
| Slack    | `slack://hook:tokenA-tokenB-tokenC@webhook`           | `slack://tokenA/tokenB/tokenC`                         |
| Telegram | `telegram://token@telegram?chats=chat_id`             | `tgram://token/chat_id`                                |
| Pushover | `pushover://shoutrrr:app_token@user_key`              | `pover://user_key@app_token`                           |
| Email    | `smtp://user:pass@host:587/?from=a@b.c&to=d@e.f`      | `mailtos://user:pass@host:587?from=a@b.c&to=d@e.f`     |
| Webhook  | `generic://host/path`                                 | `jsons://host/path` (the JSON body differs)            |

## Templates

Notifications are rendered with Go [templates](https://pkg.go.dev/text/template). Try yours in the
[template preview](template-preview.md) before deploying it.

### Simple templates

The default value if not set is `{{range .}}{{.Message}}{{println}}{{end}}`. The example below uses a template that also
outputs timestamp and log level.

!!! tip "Custom date format"
    If you want to adjust the date/time format it must show how the
    [reference time](https://pkg.go.dev/time#pkg-constants) (_Mon Jan 2 15:04:05 MST 2006_) would be displayed in your
    custom format.
    i.e., The day of the year has to be 1, the month has to be 2 (february), the hour 3 (or 15 for 24h time) etc.

!!! note "Skipping notifications"
    To skip sending notifications that do not contain any information, you can wrap your template with `{{if .}}` and `{{end}}`.

Example:

```bash
docker run -d \
  --name watchtower \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e WATCHTOWER_NOTIFICATION_URL="discord://webhook_id/webhook_token" \
  -e WATCHTOWER_NOTIFICATION_TEMPLATE="{{range .}}{{.Time.Format \"2006-01-02 15:04:05\"}} ({{.Level}}): {{.Message}}{{println}}{{end}}" \
  shounak6942/watchtower
```

### Report templates

The default template for report notifications are the following:
```go
{{- if .Report -}}
  {{- with .Report -}}
    {{- if ( or .Updated .Failed ) -}}
{{len .Scanned}} Scanned, {{len .Updated}} Updated, {{len .Failed}} Failed
      {{- range .Updated}}
- {{.Name}} ({{.ImageName}}): {{.CurrentImageID.ShortID}} updated to {{.LatestImageID.ShortID}}
      {{- end -}}
      {{- range .Fresh}}
- {{.Name}} ({{.ImageName}}): {{.State}}
	  {{- end -}}
	  {{- range .Skipped}}
- {{.Name}} ({{.ImageName}}): {{.State}}: {{.Error}}
	  {{- end -}}
	  {{- range .Failed}}
- {{.Name}} ({{.ImageName}}): {{.State}}: {{.Error}}
	  {{- end -}}
    {{- end -}}
  {{- end -}}
{{- else -}}
  {{range .Entries -}}{{.Message}}{{"\n"}}{{- end -}}
{{- end -}}
```

It will be used to send a summary of every session if there are any containers that were updated or which failed to update.

!!! note "Skipping notifications"
    Whenever the result of applying the template results in an empty string, no notifications will
    be sent. This is by default used to limit the notifications to only be sent when there something noteworthy occurred.

    You can replace `{{- if ( or .Updated .Failed ) -}}` with any logic you want to decide when to send the notifications.

Example using a custom report template that always sends a session report after each run:

=== "docker run"

    ```bash
    docker run -d \
      --name watchtower \
      -v /var/run/docker.sock:/var/run/docker.sock \
      -e WATCHTOWER_NOTIFICATION_REPORT="true" \
      -e WATCHTOWER_NOTIFICATION_URL="discord://webhook_id/webhook_token slack://token_a/token_b/token_c" \
      -e WATCHTOWER_NOTIFICATION_TEMPLATE="
      {{- if .Report -}}
        {{- with .Report -}}
      {{len .Scanned}} Scanned, {{len .Updated}} Updated, {{len .Failed}} Failed
            {{- range .Updated}}
      - {{.Name}} ({{.ImageName}}): {{.CurrentImageID.ShortID}} updated to {{.LatestImageID.ShortID}}
            {{- end -}}
            {{- range .Fresh}}
      - {{.Name}} ({{.ImageName}}): {{.State}}
          {{- end -}}
          {{- range .Skipped}}
      - {{.Name}} ({{.ImageName}}): {{.State}}: {{.Error}}
          {{- end -}}
          {{- range .Failed}}
      - {{.Name}} ({{.ImageName}}): {{.State}}: {{.Error}}
          {{- end -}}
        {{- end -}}
      {{- else -}}
        {{range .Entries -}}{{.Message}}{{\"\n\"}}{{- end -}}
      {{- end -}}
      " \
      shounak6942/watchtower
    ```

=== "compose.yml"

    ```yaml
    services:
      watchtower:
        image: shounak6942/watchtower
        volumes:
          - /var/run/docker.sock:/var/run/docker.sock
        environment:
          WATCHTOWER_NOTIFICATION_REPORT: "true"
          WATCHTOWER_NOTIFICATION_URL: >
            discord://webhook_id/webhook_token
            slack://token_a/token_b/token_c
          WATCHTOWER_NOTIFICATION_TEMPLATE: |
            {{- if .Report -}}
              {{- with .Report -}}
            {{len .Scanned}} Scanned, {{len .Updated}} Updated, {{len .Failed}} Failed
                  {{- range .Updated}}
            - {{.Name}} ({{.ImageName}}): {{.CurrentImageID.ShortID}} updated to {{.LatestImageID.ShortID}}
                  {{- end -}}
                  {{- range .Fresh}}
            - {{.Name}} ({{.ImageName}}): {{.State}}
                {{- end -}}
                {{- range .Skipped}}
            - {{.Name}} ({{.ImageName}}): {{.State}}: {{.Error}}
                {{- end -}}
                {{- range .Failed}}
            - {{.Name}} ({{.ImageName}}): {{.State}}: {{.Error}}
                {{- end -}}
              {{- end -}}
            {{- else -}}
              {{range .Entries -}}{{.Message}}{{"\n"}}{{- end -}}
            {{- end -}}
    ```

## Legacy notifications

For backwards compatibility, the notifications can also be configured using the legacy options of the original
Watchtower. They are converted to the URLs described above when Watchtower starts.
The types of notifications to send are set by passing a list of values to the `--notifications` option
(or corresponding environment variable `WATCHTOWER_NOTIFICATIONS`), which has the following valid values:

-   `email` to send notifications via e-mail
-   `slack` to send notifications through a Slack (or Discord) webhook
-   `msteams` to send notifications via a Microsoft Teams webhook or workflow
-   `gotify` to send notifications via Gotify

!!! note "Using multiple notification types with environment variables"
    Separate the values with spaces rather than commas, and do not quote the value in compose files:
    ```
    WATCHTOWER_NOTIFICATIONS=slack msteams
    ```

### `notify-upgrade`
If watchtower is started with `notify-upgrade` as it's first argument, it will generate a .env file with your current legacy notification options converted to notification URLs.

=== "docker run"

    ```bash
    $ docker run -d \
    --name watchtower \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -e WATCHTOWER_NOTIFICATIONS=slack \
    -e WATCHTOWER_NOTIFICATION_SLACK_HOOK_URL="https://hooks.slack.com/services/xxx/yyyyyyyyyyyyyyy" \
    shounak6942/watchtower \
    notify-upgrade
    ```

=== "compose.yml"

    ```yaml
    services:
      watchtower:
        image: shounak6942/watchtower
        volumes:
          - /var/run/docker.sock:/var/run/docker.sock
        environment:
          WATCHTOWER_NOTIFICATIONS: slack
          WATCHTOWER_NOTIFICATION_SLACK_HOOK_URL: https://hooks.slack.com/services/xxx/yyyyyyyyyyyyyyy
        command: notify-upgrade
    ```

You can then copy this file from the container (a message with the full command to do so will be logged) and use it with your current setup:

=== "docker run"

    ```bash
    $ docker run -d \
    --name watchtower \
    -v /var/run/docker.sock:/var/run/docker.sock \
    --env-file watchtower-notifications.env \
    shounak6942/watchtower
    ```

=== "compose.yml"

    ```yaml
    services:
      watchtower:
        image: shounak6942/watchtower
        volumes:
          - /var/run/docker.sock:/var/run/docker.sock
        env_file:
          - watchtower-notifications.env
    ```

### Email

To receive notifications by email, the following command-line options, or their corresponding environment variables, can be set:

-   `--notification-email-from` (env. `WATCHTOWER_NOTIFICATION_EMAIL_FROM`): The e-mail address from which notifications will be sent.
-   `--notification-email-to` (env. `WATCHTOWER_NOTIFICATION_EMAIL_TO`): The e-mail address to which notifications will be sent.
-   `--notification-email-server` (env. `WATCHTOWER_NOTIFICATION_EMAIL_SERVER`): The SMTP server to send e-mails through.
-   `--notification-email-server-tls-skip-verify` (env. `WATCHTOWER_NOTIFICATION_EMAIL_SERVER_TLS_SKIP_VERIFY`): Do not verify the TLS certificate of the mail server. This should be used only for testing.
-   `--notification-email-server-port` (env. `WATCHTOWER_NOTIFICATION_EMAIL_SERVER_PORT`): The port used to connect to the SMTP server to send e-mails through. Defaults to `25`, which uses STARTTLS when the server offers it. Other ports require TLS.
-   `--notification-email-server-user` (env. `WATCHTOWER_NOTIFICATION_EMAIL_SERVER_USER`): The username to authenticate with the SMTP server with.
-   `--notification-email-server-password` (env. `WATCHTOWER_NOTIFICATION_EMAIL_SERVER_PASSWORD`): The password to authenticate with the SMTP server with. Can also reference a file, in which case the contents of the file are used.
-   `--notification-email-delay` (env. `WATCHTOWER_NOTIFICATION_EMAIL_DELAY`): Delay before sending notifications expressed in seconds.
-   `--notification-email-subjecttag` (env. `WATCHTOWER_NOTIFICATION_EMAIL_SUBJECTTAG`): Prefix to include in the subject tag. Useful when running multiple watchtowers. **NOTE:** This will affect all notification types.

Example:

```bash
docker run -d \
  --name watchtower \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e WATCHTOWER_NOTIFICATIONS=email \
  -e WATCHTOWER_NOTIFICATION_EMAIL_FROM=fromaddress@gmail.com \
  -e WATCHTOWER_NOTIFICATION_EMAIL_TO=toaddress@gmail.com \
  -e WATCHTOWER_NOTIFICATION_EMAIL_SERVER=smtp.gmail.com \
  -e WATCHTOWER_NOTIFICATION_EMAIL_SERVER_PORT=587 \
  -e WATCHTOWER_NOTIFICATION_EMAIL_SERVER_USER=fromaddress@gmail.com \
  -e WATCHTOWER_NOTIFICATION_EMAIL_SERVER_PASSWORD=app_password \
  -e WATCHTOWER_NOTIFICATION_EMAIL_DELAY=2 \
  shounak6942/watchtower
```

The equivalent notification URL is
`mailtos://fromaddress%40gmail.com:app_password@smtp.gmail.com:587?to=toaddress@gmail.com`.

### Slack

To receive notifications in Slack, add `slack` to the `--notifications` option or the `WATCHTOWER_NOTIFICATIONS` environment variable.

Additionally, you should set the Slack webhook URL using the `--notification-slack-hook-url` option or the `WATCHTOWER_NOTIFICATION_SLACK_HOOK_URL` environment variable. This option can also reference a file, in which case the contents of the file are used. Discord webhook URLs are accepted too.

By default, watchtower will send messages under the name `watchtower`, you can customize this string through the `--notification-slack-identifier` option or the `WATCHTOWER_NOTIFICATION_SLACK_IDENTIFIER` environment variable.

Other, optional, variables include:

-   `--notification-slack-channel` (env. `WATCHTOWER_NOTIFICATION_SLACK_CHANNEL`): A string which overrides the webhook's default channel. Example: #my-custom-channel.
-   `--notification-slack-icon-emoji` (env. `WATCHTOWER_NOTIFICATION_SLACK_ICON_EMOJI`) and `--notification-slack-icon-url` (env. `WATCHTOWER_NOTIFICATION_SLACK_ICON_URL`): The icon of the messages.

Example:

```bash
docker run -d \
  --name watchtower \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e WATCHTOWER_NOTIFICATIONS=slack \
  -e WATCHTOWER_NOTIFICATION_SLACK_HOOK_URL="https://hooks.slack.com/services/xxx/yyyyyyyyy/zzzzzzzzzzzzzzzzzzzzzzzz" \
  -e WATCHTOWER_NOTIFICATION_SLACK_IDENTIFIER=watchtower-server-1 \
  -e WATCHTOWER_NOTIFICATION_SLACK_CHANNEL=#my-custom-channel \
  shounak6942/watchtower
```

### Microsoft Teams

To receive notifications in a Microsoft Teams channel, add `msteams` to the `--notifications` option or the `WATCHTOWER_NOTIFICATIONS` environment variable.

Additionally, set the webhook URL using the `--notification-msteams-hook` option or the `WATCHTOWER_NOTIFICATION_MSTEAMS_HOOK_URL` environment variable. This option can also reference a file, in which case the contents of the file are used. Both Power Automate workflow URLs and the retired Office 365 connector URLs are accepted.

-   `--notification-msteams-data` (env. `WATCHTOWER_NOTIFICATION_MSTEAMS_USE_LOG_DATA`): Legacy option for including log entry fields as message facts. It is accepted for compatibility, but has no effect.

Example:

```bash
docker run -d \
  --name watchtower \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e WATCHTOWER_NOTIFICATIONS=msteams \
  -e WATCHTOWER_NOTIFICATION_MSTEAMS_HOOK_URL="https://prod-00.westus.logic.azure.com:443/workflows/xxxxxxxx/triggers/manual/paths/invoke?api-version=2016-06-01&sp=%2Ftriggers%2Fmanual%2Frun&sv=1.0&sig=yyyyyyyy" \
  shounak6942/watchtower
```

### Gotify

To push a notification to your Gotify instance, add `gotify` to the `--notifications` option and set:

-   `--notification-gotify-url` (env. `WATCHTOWER_NOTIFICATION_GOTIFY_URL`): The base URL of the Gotify server, starting with `http://` or `https://`.
-   `--notification-gotify-token` (env. `WATCHTOWER_NOTIFICATION_GOTIFY_TOKEN`): The application token. Can also reference a file, in which case the contents of the file are used.
-   `--notification-gotify-tls-skip-verify` (env. `WATCHTOWER_NOTIFICATION_GOTIFY_TLS_SKIP_VERIFY`): Do not verify the TLS certificate of the server. This should be used only for testing.

Example:

```bash
docker run -d \
  --name watchtower \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e WATCHTOWER_NOTIFICATIONS=gotify \
  -e WATCHTOWER_NOTIFICATION_GOTIFY_URL="https://gotify.example.com/" \
  -e WATCHTOWER_NOTIFICATION_GOTIFY_TOKEN="AbCdEfGhIjKlMnO" \
  shounak6942/watchtower
```
