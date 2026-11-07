When watchtower detects that a running container needs to be updated it will stop the container by sending it the
stop signal configured for the container, using `STOPSIGNAL` in its Dockerfile or `--stop-signal` when it was started, or
SIGTERM when none is configured. Containers that have not stopped when the [stop timeout](arguments.md#wait_until_timeout)
expires are killed.

If your container should be shutdown with a different signal you can communicate this to watchtower by setting a label named _com.centurylinklabs.watchtower.stop-signal_ with the value of the desired signal.

This label can be coded directly into your image by using the `LABEL` instruction in your Dockerfile:

```docker
LABEL com.centurylinklabs.watchtower.stop-signal="SIGHUP"
```

Or, it can be specified as part of the `docker run` command line:

```bash
docker run -d --label=com.centurylinklabs.watchtower.stop-signal=SIGHUP someimage
```
