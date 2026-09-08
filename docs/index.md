<p style="text-align: center; margin-left: 1.6rem;">
  <img alt="Logotype depicting a lighthouse" src="./images/logo-450px.png" width="450" />
</p>
<h1 align="center">
  Watchtower
</h1>

<p align="center">
  A container-based solution for automating Docker container base image updates.
  <br/><br/>
  <a href="https://github.com/dallergy/watchtower/actions/workflows/pull-request.yml">
    <img alt="CI" src="https://github.com/dallergy/watchtower/actions/workflows/pull-request.yml/badge.svg" />
  </a>
  <a href="https://codecov.io/gh/dallergy/watchtower">
    <img alt="Codecov" src="https://codecov.io/gh/dallergy/watchtower/branch/main/graph/badge.svg">
  </a>
  <a href="https://godoc.org/github.com/containrrr/watchtower">
    <img alt="GoDoc" src="https://godoc.org/github.com/containrrr/watchtower?status.svg" />
  </a>
  <a href="https://goreportcard.com/report/github.com/containrrr/watchtower">
    <img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/containrrr/watchtower" />
  </a>
  <a href="https://github.com/dallergy/watchtower/releases">
    <img alt="latest version" src="https://img.shields.io/github/tag/dallergy/watchtower.svg" />
  </a>
  <a href="https://www.apache.org/licenses/LICENSE-2.0">
    <img alt="Apache-2.0 License" src="https://img.shields.io/github/license/dallergy/watchtower.svg" />
  </a>
  <a href="https://github.com/containrrr/watchtower/#contributors">
    <img alt="All Contributors" src="https://img.shields.io/github/all-contributors/containrrr/watchtower" />
  </a>
  <a href="https://hub.docker.com/r/shounak6942/watchtower">
    <img alt="Pulls from DockerHub" src="https://img.shields.io/docker/pulls/shounak6942/watchtower.svg" />
  </a>
</p>

!!! note "Community-maintained fork"
    The original [containrrr/watchtower](https://github.com/containrrr/watchtower) project is no longer maintained.
    This fork is actively maintained at [dallergy/watchtower](https://github.com/dallergy/watchtower) and published to Docker Hub as `shounak6942/watchtower`.

## Quick Start

With watchtower you can update the running version of your containerized app simply by pushing a new image to the Docker
Hub or your own image registry. Watchtower will pull down your new image, gracefully shut down your existing container
and restart it with the same options that were used when it was deployed initially. Run the watchtower container with
the following command:

=== "docker run"

    ```bash
    $ docker run -d \
    --name watchtower \
    -v /var/run/docker.sock:/var/run/docker.sock \
    shounak6942/watchtower
    ```

=== "docker-compose.yml"

    ```yaml
    services:
      watchtower:
        image: shounak6942/watchtower
        restart: unless-stopped
        volumes:
          - /var/run/docker.sock:/var/run/docker.sock:ro
        environment:
          WATCHTOWER_CLEANUP: "true"
    ```

Notifications use Apprise bundled in the image. See [Compose and environment variables](compose.md) and [Notifications](notifications.md).
