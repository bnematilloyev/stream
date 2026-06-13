# Official image has no shell/curl — runOnReady hooks need curl for media-orchestrator.
FROM bluenviron/mediamtx:latest AS mediamtx
FROM alpine:3.20

RUN apk add --no-cache curl ca-certificates

COPY --from=mediamtx /mediamtx /mediamtx

ENTRYPOINT ["/mediamtx"]
