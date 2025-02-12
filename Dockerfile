FROM golang:1.24.0-alpine AS builder
RUN apk add --update --no-cache ca-certificates git

FROM scratch
COPY peribolosmerger /
COPY --from=builder /etc/ssl/certs /etc/ssl/certs

ENTRYPOINT ["/peribolosmerger"]
