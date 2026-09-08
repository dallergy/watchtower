FROM alpine:3.21

LABEL "com.centurylinklabs.watchtower"="true"

RUN apk add --no-cache \
        ca-certificates \
        tzdata \
        python3 \
        py3-pip \
    && pip3 install --no-cache-dir --break-system-packages "apprise==1.9.3" \
    && apk del --no-network py3-pip \
    && find /usr -type d -name '__pycache__' -prune -exec rm -rf {} + || true \
    && rm -rf /root/.cache /tmp/* /var/cache/apk/*

EXPOSE 8080

COPY watchtower /watchtower

HEALTHCHECK CMD [ "/watchtower", "--health-check"]

ENTRYPOINT ["/watchtower"]
