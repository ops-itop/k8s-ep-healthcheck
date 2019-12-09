FROM alpine:3.10
COPY bin/ep-healthcheck /ep-healthcheck
ENTRYPOINT /ep-healthcheck
