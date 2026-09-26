# Contributing

## Prerequisites

To contribute code changes to this project you will need:

- [Go](https://go.dev/doc/install) 1.26 or later. With an older toolchain installed, `go` downloads the version
  listed in [go.mod](go.mod) automatically.
- [Docker](https://docs.docker.com/engine/install/) with Buildx, to build images.

## Checking out the code

```bash
git clone https://github.com/dallergy/watchtower.git
cd watchtower
```

## Building and testing

Watchtower is a Go application. The following commands assume that you are at the root of the repository:

```bash
go build                           # builds the watchtower binary
go test -race ./...                # runs the tests
./watchtower                       # runs the application (outside of a container)
```

Pull requests are also checked with the following tools, which are worth running before pushing:

```bash
golangci-lint run                  # lint and formatting (golangci-lint v2)
govulncheck ./...                  # known vulnerabilities in reachable code
```

## Building images

The Dockerfiles live in `dockerfiles/`:

- `Dockerfile.dev-self-contained` builds an image from your local checkout.
- `Dockerfile.self-contained` builds an image from a branch or tag of this repository on GitHub.
- `Dockerfile` packages binaries built by GoReleaser and is used for releases.

The runtime image is based on `scratch`: it only contains CA certificates and the static Watchtower binary, which
embeds the time zone database. Notifications are sent by Watchtower itself, so nothing else is needed.

```bash
docker build -f dockerfiles/Dockerfile.dev-self-contained -t watchtower .

# multi-platform, without emulation
docker buildx build -f dockerfiles/Dockerfile.dev-self-contained \
  --platform linux/amd64,linux/arm64,linux/arm/v7 -t watchtower .
```

## Documentation

The documentation site is built with [Material for MkDocs](https://squidfunk.github.io/mkdocs-material/):

```bash
scripts/build-tplprev.sh           # builds the template preview (WebAssembly)
pip install -r docs-requirements.txt
mkdocs serve
```

## Releasing

- Every push to `main` publishes `latest`, `latest-dev` and `sha-<commit>` images to Docker Hub
  (`shounak6942/watchtower`) and GHCR (`ghcr.io/dallergy/watchtower`), after the image passed a vulnerability scan.
- Pushing a `vX.Y.Z` tag runs GoReleaser, which publishes the GitHub release with binaries and the versioned images.

Publishing to Docker Hub needs the repository secrets `DOCKER_USERNAME` and `DOCKER_PASSWORD`, where the password is a
Docker Hub [access token](https://docs.docker.com/security/for-developers/access-tokens/) with read and write access.
GHCR uses the workflow's `GITHUB_TOKEN`. If pulling from GHCR is denied, make the package public in its
[package settings](https://github.com/users/dallergy/packages/container/package/watchtower).
