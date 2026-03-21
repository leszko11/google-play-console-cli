# Stage 1: Build
FROM golang:1.24-alpine AS builder

ARG TARGETOS=linux
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w \
      -X 'github.com/leszko11/google-play-console-cli/cmd.Version=${VERSION}' \
      -X 'github.com/leszko11/google-play-console-cli/cmd.Commit=${COMMIT}' \
      -X 'github.com/leszko11/google-play-console-cli/cmd.Date=${DATE}'" \
    -o /gpc .

# Stage 2: Runtime
FROM alpine:3.21

LABEL maintainer="leszko11"
LABEL org.opencontainers.image.source="https://github.com/leszko11/google-play-console-cli"
LABEL org.opencontainers.image.description="Google Play Console CLI"

RUN apk add --no-cache ca-certificates \
    && addgroup -S gpc && adduser -S gpc -G gpc

COPY --from=builder /gpc /usr/local/bin/gpc

USER gpc

ENTRYPOINT ["gpc"]
