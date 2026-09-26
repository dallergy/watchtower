# Runtime image for the binaries built by GoReleaser (see goreleaser.yml).
#
# Watchtower is a static binary with the time zone database embedded, so the image only adds
# CA certificates on top of an empty base: there are no OS packages to patch.
FROM --platform=$BUILDPLATFORM alpine:3.24 AS certs

FROM scratch
ARG TARGETPLATFORM

LABEL "com.centurylinklabs.watchtower"="true"

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY $TARGETPLATFORM/watchtower /watchtower

EXPOSE 8080

HEALTHCHECK CMD ["/watchtower", "--health-check"]

ENTRYPOINT ["/watchtower"]
