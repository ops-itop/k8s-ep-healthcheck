FROM alpine:3.10
COPY templates /templates
COPY bin/ep-healthcheck /ep-healthcheck
ENTRYPOINT /ep-healthcheck
