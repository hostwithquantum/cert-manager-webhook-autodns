FROM alpine:3.23

RUN apk add --no-cache ca-certificates

ADD webhook /usr/local/bin/webhook

ENTRYPOINT ["webhook"]
