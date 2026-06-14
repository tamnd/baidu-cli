FROM alpine:3.21

ARG TARGETPLATFORM

RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -H -u 10001 baidu

COPY $TARGETPLATFORM/baidu /usr/bin/baidu

USER baidu

ENTRYPOINT ["/usr/bin/baidu"]
