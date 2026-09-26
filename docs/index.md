---
hide:
  - navigation
  - toc
---

<div class="wt-hero" markdown>

<img class="wt-hero__logo" src="images/logo-450px.png" alt="Watchtower logo" width="80" height="80">

# Keep your containers up to date, automatically

<p class="wt-hero__lead">
Watchtower watches your running containers. When a newer image is pushed, it pulls it and
restarts the container with the exact options it was started with.
</p>

<div class="wt-hero__actions">
  <a class="md-button md-button--primary" href="#quick-start">Get started</a>
  <a class="md-button" href="notifications/">Set up notifications</a>
  <a class="md-button" href="https://github.com/dallergy/watchtower">GitHub</a>
</div>

<ul class="wt-pills">
  <li>Scratch image, no OS packages</li>
  <li>amd64 · arm64 · armv7 · armv6 · 386</li>
  <li>Docker Hub &amp; GHCR</li>
</ul>

</div>

## Quick start

Run Watchtower next to your other containers and give it access to the Docker socket:

=== "docker run"

    ```bash
    docker run -d \
      --name watchtower \
      --restart unless-stopped \
      -v /var/run/docker.sock:/var/run/docker.sock \
      shounak6942/watchtower
    ```

=== "compose.yml"

    ```yaml
    services:
      watchtower:
        image: shounak6942/watchtower
        restart: unless-stopped
        volumes:
          - /var/run/docker.sock:/var/run/docker.sock
        environment:
          WATCHTOWER_CLEANUP: "true"
    ```

The same image is published as `ghcr.io/dallergy/watchtower`. See [Compose and environment](compose.md) for a
complete, commented setup.

## What you get

<div class="grid cards" markdown>

-   :material-shield-check-outline:{ .lg .middle } __Small and secure__

    ---

    A static binary on an empty base image: a few megabytes, and no shell or OS packages that
    could carry vulnerabilities.

-   :material-bell-ring-outline:{ .lg .middle } __Built-in notifications__

    ---

    Gotify, ntfy, Slack, Discord, Telegram, email, Teams, Pushover and webhooks, configured with
    Apprise-style URLs. No sidecar container needed.

    [:octicons-arrow-right-24: Notifications](notifications.md)

-   :material-filter-variant:{ .lg .middle } __You decide what updates__

    ---

    Opt containers in or out with labels, monitor without updating, or scope instances to
    groups of containers.

    [:octicons-arrow-right-24: Container selection](container-selection.md)

-   :material-api:{ .lg .middle } __Automation friendly__

    ---

    Trigger updates from CI through the HTTP API, run lifecycle hooks around updates and export
    Prometheus metrics.

    [:octicons-arrow-right-24: HTTP API mode](http-api-mode.md)

</div>

!!! note "A maintained fork"
    The original [containrrr/watchtower](https://github.com/containrrr/watchtower) project is no longer
    maintained. This fork keeps Watchtower current with the latest Docker Engine releases, dependencies and
    security fixes. Switching is usually just a matter of replacing `containrrr/watchtower` with
    `shounak6942/watchtower`, but notification URLs now use Apprise instead of Shoutrrr syntax, see
    [migrating notifications](notifications.md#migrating_from_shoutrrr).
