FROM alpine:3.23

ARG TARGETPLATFORM

RUN apk add --no-cache ca-certificates

ADD ${TARGETPLATFORM}/webhook /usr/local/bin/webhook

ENTRYPOINT ["webhook"]