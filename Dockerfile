FROM alpine:3.24.1

ARG TARGETPLATFORM

RUN apk add --no-cache ca-certificates

ADD ${TARGETPLATFORM}/webhook /usr/local/bin/webhook

ENTRYPOINT ["webhook"]