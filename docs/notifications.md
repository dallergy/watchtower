# Notifications

Watchtower sends notifications when containers are updated. There is **no second Apprise container**. Gotify is built into Watchtower. Other services (Discord, Telegram, email, …) use Apprise that is already inside the same image.

Notifications are triggered from [logrus](http://github.com/sirupsen/logrus) hooks.

## Gotify (built in)

Gotify is sent directly to your Gotify server. There is no Apprise service to run.

Either of these is enough:

=== "Gotify URL"

    ```yaml
    environment:
      WATCHTOWER_CLEANUP: "true"
      WATCHTOWER_POLL_INTERVAL: "7200"
      WATCHTOWER_NOTIFICATION_URL: gotifys://gotify.example.com/your.gotify.application.token
    ```

    `gotify://host/token` on a public hostname also uses HTTPS. Prefer `gotifys://` for TLS servers.

Flags: `--notifications`, `--notification-gotify-url`, `--notification-gotify-token`, `--notification-gotify-tls-skip-verify` (env `WATCHTOWER_NOTIFICATION_GOTIFY_TLS_SKIP_VERIFY`).

=== "Gotify env vars"

    ```yaml
    environment:
      WATCHTOWER_NOTIFICATIONS: gotify
      WATCHTOWER_NOTIFICATIONS_LEVEL: info
      WATCHTOWER_NOTIFICATION_GOTIFY_URL: https://gotify.example.com/
      WATCHTOWER_NOTIFICATION_GOTIFY_TOKEN: your.gotify.application.token
    ```

=== "docker-compose"

    ```yaml
    services:
      watchtower:
        image: shounak6942/watchtower:latest
        restart: unless-stopped
        volumes:
          - /var/run/docker.sock:/var/run/docker.sock:ro
        environment:
          WATCHTOWER_CLEANUP: "true"
          WATCHTOWER_NOTIFICATION_REPORT: "true"
          WATCHTOWER_NO_STARTUP_MESSAGE: "true"
          WATCHTOWER_NOTIFICATIONS: gotify
          WATCHTOWER_NOTIFICATIONS_LEVEL: info
          WATCHTOWER_NOTIFICATION_GOTIFY_URL: https://gotify.example.com/
          WATCHTOWER_NOTIFICATION_GOTIFY_TOKEN: your.gotify.application.token
    ```

=== "docker run"

    ```bash
    docker run -d \
      --name watchtower \
      --restart unless-stopped \
      -v /var/run/docker.sock:/var/run/docker.sock \
      -e WATCHTOWER_CLEANUP=true \
      -e WATCHTOWER_NOTIFICATIONS=gotify \
      -e WATCHTOWER_NOTIFICATIONS_LEVEL=info \
      -e WATCHTOWER_NOTIFICATION_GOTIFY_URL=https://gotify.example.com/ \
      -e WATCHTOWER_NOTIFICATION_GOTIFY_TOKEN=your.gotify.application.token \
      shounak6942/watchtower
    ```

See [Compose and environment variables](compose.md) for the full env list.

## Settings

-   `--notifications-level` (env. `WATCHTOWER_NOTIFICATIONS_LEVEL`): Controls the log level which is used for the notifications. If omitted, the default log level is `info`. Possible values are: `panic`, `fatal`, `error`, `warn`, `info`, `debug` or `trace`.
-   `--notifications-hostname` (env. `WATCHTOWER_NOTIFICATIONS_HOSTNAME`): Custom hostname specified in subject/title. Useful to override the operating system hostname.
-   `--notifications-delay` (env. `WATCHTOWER_NOTIFICATIONS_DELAY`): Delay before sending notifications expressed in seconds.
-   Watchtower will post a notification every time it is started. This behavior [can be changed](arguments.md#without_sending_a_startup_message) with an argument.
-   `--notification-title-tag` (env. `WATCHTOWER_NOTIFICATION_TITLE_TAG`): Prefix to include in the title. Useful when running multiple watchtowers.
-   `--notification-skip-title` (env. `WATCHTOWER_NOTIFICATION_SKIP_TITLE`): Do not pass the title param to notifications. This will not pass a dynamic title override to notification services. If no title is configured for the service, it will remove the title all together.
-   `--notification-log-stdout` (env. `WATCHTOWER_NOTIFICATION_LOG_STDOUT`): Write rendered notification output to stdout.

## Other services (built-in Apprise)

!!! note "Using multiple notifications with environment variables"
    There is currently a bug in Viper (https://github.com/spf13/viper/issues/380), which prevents comma-separated slices to
    be used when using the environment variable.  
    A workaround is available where we instead put quotes around the environment variable value and replace the commas with
    spaces:
    ```
    WATCHTOWER_NOTIFICATIONS="slack msteams"
    ```
    If you're a `docker-compose` user, make sure to specify environment variables' values in your `.yml` file without double
    quotes (`"`). This prevents unexpected errors when watchtower starts.

Set one or more [Apprise service URLs](https://github.com/caronc/apprise/wiki). Watchtower calls the `apprise` CLI that is already in the image.

-   `--notification-url` (env. `WATCHTOWER_NOTIFICATION_URL`): Apprise-compatible service URL(s). This option can also reference a file, in which case the contents of the file are used.
-   `--notification-apprise-config` (env. `WATCHTOWER_NOTIFICATION_APPRISE_CONFIG`): Optional path *inside the container* to an [Apprise configuration file](https://github.com/caronc/apprise/wiki/config). Mount the file as a volume.
-   `--notification-template` (env. `WATCHTOWER_NOTIFICATION_TEMPLATE`): Go [template](https://golang.org/pkg/text/template/) used for the message.
-   `--notification-report` (env. `WATCHTOWER_NOTIFICATION_REPORT`): Use the session report as the notification template data.

You can define multiple services by space-separating the URLs.

### Example (single container)

=== "docker run"

    ```bash
    docker run -d \
      --name watchtower \
      --restart unless-stopped \
      -v /var/run/docker.sock:/var/run/docker.sock \
      -e WATCHTOWER_CLEANUP=true \
      -e WATCHTOWER_NOTIFICATION_REPORT=true \
      -e WATCHTOWER_NO_STARTUP_MESSAGE=true \
      -e WATCHTOWER_NOTIFICATION_URL="discord://webhook_id/webhook_token tgram://bot_token/chat_id" \
      shounak6942/watchtower
    ```

=== "docker-compose"

    ```yaml
    services:
      watchtower:
        image: shounak6942/watchtower:latest
        restart: unless-stopped
        volumes:
          - /var/run/docker.sock:/var/run/docker.sock:ro
        environment:
          WATCHTOWER_CLEANUP: "true"
          WATCHTOWER_NOTIFICATION_REPORT: "true"
          WATCHTOWER_NO_STARTUP_MESSAGE: "true"
          WATCHTOWER_NOTIFICATION_URL: discord://webhook_id/webhook_token
    ```

See [Compose and environment variables](compose.md) for a full env reference.

## Simple templates

The default value if not set is `{{range .}}{{.Message}}{{println}}{{end}}`. The example below uses a template that also
outputs timestamp and log level.

!!! tip "Custom date format"
    If you want to adjust the date/time format it must show how the
    [reference time](https://golang.org/pkg/time/#pkg-constants) (_Mon Jan 2 15:04:05 MST 2006_) would be displayed in your
    custom format.  
    i.e., The day of the year has to be 1, the month has to be 2 (february), the hour 3 (or 15 for 24h time) etc.

!!! note "Skipping notifications"
    To skip sending notifications that do not contain any information, you can wrap your template with `{{if .}}` and `{{end}}`.

Example:

```bash
docker run -d \
  --name watchtower \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e WATCHTOWER_NOTIFICATION_URL="discord://token/webhook_id" \
  -e WATCHTOWER_NOTIFICATION_TEMPLATE="{{range .}}{{.Time.Format \"2006-01-02 15:04:05\"}} ({{.Level}}): {{.Message}}{{println}}{{end}}" \
  shounak6942/watchtower
```

## Report templates

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
      -e WATCHTOWER_NOTIFICATION_URL="discord://token/webhook_id slack://token_a/token_b/token_c" \
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

=== "docker-compose"

    ```yaml
    services:
      watchtower:
        image: shounak6942/watchtower
        volumes:
          - /var/run/docker.sock:/var/run/docker.sock
        environment:
          WATCHTOWER_NOTIFICATION_REPORT: "true"
          WATCHTOWER_NOTIFICATION_URL: >
            discord://token/webhook_id
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

For backwards compatibility, the notifications can also be configured using legacy notification options. These will automatically be converted to Apprise-compatible URLs when used.  
The types of notifications to send are set by passing a comma-separated list of values to the `--notifications` option
(or corresponding environment variable `WATCHTOWER_NOTIFICATIONS`), which has the following valid values:

-   `email` to send notifications via e-mail
-   `slack` to send notifications through a Slack webhook
-   `msteams` to send notifications via MSTeams webhook
-   `gotify` to send notifications via Gotify

### `notify-upgrade`
If watchtower is started with `notify-upgrade` as it's first argument, it will generate a .env file with your current legacy notification options converted to Apprise URLs.

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

=== "docker-compose.yml"

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

=== "docker-compose.yml"

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
-   `--notification-email-server-port` (env. `WATCHTOWER_NOTIFICATION_EMAIL_SERVER_PORT`): The port used to connect to the SMTP server to send e-mails through. Defaults to `25`.
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

You can also use a native Apprise mail URL instead of the legacy flags, for example `mailto://user:app_password@gmail.com`.

### Slack

To receive notifications in Slack, add `slack` to the `--notifications` option or the `WATCHTOWER_NOTIFICATIONS` environment variable.

Additionally, you should set the Slack webhook URL using the `--notification-slack-hook-url` option or the `WATCHTOWER_NOTIFICATION_SLACK_HOOK_URL` environment variable. This option can also reference a file, in which case the contents of the file are used.

By default, watchtower will send messages under the name `watchtower`, you can customize this string through the `--notification-slack-identifier` option or the `WATCHTOWER_NOTIFICATION_SLACK_IDENTIFIER` environment variable.

Other, optional, variables include:

-   `--notification-slack-channel` (env. `WATCHTOWER_NOTIFICATION_SLACK_CHANNEL`): A string which overrides the webhook's default channel. Example: #my-custom-channel.

Example:

```bash
docker run -d \
  --name watchtower \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e WATCHTOWER_NOTIFICATIONS=slack \
  -e WATCHTOWER_NOTIFICATION_SLACK_HOOK_URL="https://hooks.slack.com/services/xxx/yyyyyyyyyyyyyyy" \
  -e WATCHTOWER_NOTIFICATION_SLACK_IDENTIFIER=watchtower-server-1 \
  -e WATCHTOWER_NOTIFICATION_SLACK_CHANNEL=#my-custom-channel \
  shounak6942/watchtower
```

### Microsoft Teams

To receive notifications in MSTeams channel, add `msteams` to the `--notifications` option or the `WATCHTOWER_NOTIFICATIONS` environment variable.

Additionally, you should set the MSTeams webhook URL using the `--notification-msteams-hook` option or the `WATCHTOWER_NOTIFICATION_MSTEAMS_HOOK_URL` environment variable. This option can also reference a file, in which case the contents of the file are used.

-   `--notification-msteams-data` (env. `WATCHTOWER_NOTIFICATION_MSTEAMS_USE_LOG_DATA`): Legacy option for including log entry fields as MSTeams message facts. This option is retained for compatibility but has limited effect when using Apprise.

Example:

```bash
docker run -d \
  --name watchtower \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e WATCHTOWER_NOTIFICATIONS=msteams \
  -e WATCHTOWER_NOTIFICATION_MSTEAMS_HOOK_URL="https://outlook.office.com/webhook/xxxxxxxx@xxxxxxx/IncomingWebhook/yyyyyyyy/zzzzzzzzzz" \
  shounak6942/watchtower
```

### Gotify

Gotify is documented at the top of this page. Use `WATCHTOWER_NOTIFICATIONS=gotify` plus `WATCHTOWER_NOTIFICATION_GOTIFY_URL` and `WATCHTOWER_NOTIFICATION_GOTIFY_TOKEN`.
