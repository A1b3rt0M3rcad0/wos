FROM golang:1.27.1-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
# Optional build-only trust bundle for an HTTPS package proxy.
RUN --mount=type=secret,id=go-ca-bundle \
    if [ -r /run/secrets/go-ca-bundle ]; then \
      SSL_CERT_FILE=/run/secrets/go-ca-bundle go mod download; \
    else go mod download; fi
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags='-s -w' -o /wos ./packages/wos-api/cmd/wos

FROM alpine:3.23
# The pinned official Go build image already supplies the CA bundle.
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
RUN addgroup -g 10001 wos && adduser -D -u 10001 -G wos wos && mkdir /data && chown wos:wos /data
COPY --from=build /wos /usr/local/bin/wos
USER 10001:10001
ENV WOS_SQLITE_PATH=/data/wos.db
EXPOSE 8080
ENTRYPOINT ["wos"]
CMD ["server"]
