FROM golang:1.27.1-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
# Optional build-only trust bundle for an HTTPS package proxy.
RUN --mount=type=secret,id=go-ca-bundle \
    if [ -r /run/secrets/go-ca-bundle ]; then \
      SSL_CERT_FILE=/run/secrets/go-ca-bundle go mod download; \
    else go mod download; fi
COPY . .
ARG WOS_VERSION=dev
ARG WOS_COMMIT=unknown
ARG WOS_BUILT_AT=unknown
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false \
    -ldflags="-s -w -X github.com/A1b3rt0M3rcad0/wos/packages/wos-api/internal/server.Version=${WOS_VERSION} -X github.com/A1b3rt0M3rcad0/wos/packages/wos-api/internal/server.Commit=${WOS_COMMIT} -X github.com/A1b3rt0M3rcad0/wos/packages/wos-api/internal/server.BuiltAt=${WOS_BUILT_AT}" \
    -o /wos ./packages/wos-api/cmd/wos \
    && CGO_ENABLED=0 go build -trimpath -buildvcs=false \
    -ldflags="-s -w -X github.com/A1b3rt0M3rcad0/wos/packages/wos-cli.Version=${WOS_VERSION} -X github.com/A1b3rt0M3rcad0/wos/packages/wos-cli.Commit=${WOS_COMMIT} -X github.com/A1b3rt0M3rcad0/wos/packages/wos-cli.BuiltAt=${WOS_BUILT_AT}" \
    -o /wosctl ./packages/wos-cli/cmd/wosctl \
    && sh tools/distribution/notices.sh /licenses

FROM alpine:3.23
ARG WOS_VERSION=dev
ARG WOS_COMMIT=unknown
ARG WOS_BUILT_AT=unknown
LABEL org.opencontainers.image.title="WOS" \
      org.opencontainers.image.source="https://github.com/A1b3rt0M3rcad0/wos" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.version="${WOS_VERSION}" \
      org.opencontainers.image.revision="${WOS_COMMIT}" \
      org.opencontainers.image.created="${WOS_BUILT_AT}"
# The pinned official Go build image already supplies the CA bundle.
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
RUN addgroup -g 10001 wos && adduser -D -u 10001 -G wos wos && mkdir /data && chown wos:wos /data
COPY --from=build /wos /usr/local/bin/wos
COPY --from=build /wosctl /usr/local/bin/wosctl
COPY --from=build /licenses /usr/share/doc/wos/third-party-notices
COPY LICENSE /usr/share/doc/wos/LICENSE
USER 10001:10001
ENV WOS_SQLITE_PATH=/data/wos.db
EXPOSE 8080
ENTRYPOINT ["wos"]
CMD ["server"]
