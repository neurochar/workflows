ARG GO_VERSION=1.25.7

FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src

RUN apk add --no-cache git ca-certificates && update-ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION="dev"

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -tags timetzdata -buildvcs=false \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/app ./cmd/backend

FROM alpine:3.20 AS release
WORKDIR /app

RUN addgroup -S app && adduser -S -G app app \
    && apk add --no-cache ca-certificates \
    && update-ca-certificates

COPY --from=build /out/app ./app
COPY --from=build /src/configs ./configs
COPY --from=build /src/migrations ./migrations
COPY --from=build /src/swagger ./swagger

RUN chown -R app:app /app \
    && chmod 555 ./app

USER app
ENTRYPOINT ["./app"]