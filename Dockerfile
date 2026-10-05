FROM golang:1.27.1-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags='-s -w' -o /wos ./packages/wos-api/cmd/wos

FROM alpine:3.23
RUN apk add --no-cache ca-certificates && addgroup -g 10001 wos && adduser -D -u 10001 -G wos wos && mkdir /data && chown wos:wos /data
COPY --from=build /wos /usr/local/bin/wos
USER 10001:10001
ENV WOS_SQLITE_PATH=/data/wos.db
EXPOSE 8080
ENTRYPOINT ["wos"]
CMD ["server"]
