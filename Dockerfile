FROM alpine:3.22

RUN apk add --no-cache ca-certificates

ADD webhook /usr/local/bin/webhook

ENTRYPOINT ["webhook"]
