# syntax=docker/dockerfile:1.7
# Multi-stage build with metadata for Shipt Go Hello World app

############################
# Build stage
############################
ARG REGISTRY_OVERRIDE=docker.io
ARG GO_VERSION=1.25.3
FROM ${REGISTRY_OVERRIDE}/golang:${GO_VERSION}-alpine AS build

ARG APP_NAME="go-helloworld"
ARG VERSION="dev"
ARG COMMIT_SHA="unknown"
ARG BUILD_TIME="1970-01-01T00:00:00Z"
ARG BUILDPLATFORM
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

ENV CGO_ENABLED=0 \
    GOOS=${TARGETOS:-linux} \
    GOARCH=${TARGETARCH:-amd64} \
    GOTOOLCHAIN=auto \
    GOFLAGS="-trimpath -mod=readonly"

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /out/app \
      -ldflags "-s -w -buildid= -X 'main.version=${VERSION}' -X 'main.commit=${COMMIT_SHA}' -X 'main.buildTime=${BUILD_TIME}'" \
      .

############################
# Runtime stage
############################
FROM gcr.io/distroless/static:nonroot

ARG APP_NAME \
    VERSION \
    COMMIT_SHA \
    BUILD_TIME
LABEL org.opencontainers.image.title="${APP_NAME}" \
      org.opencontainers.image.description="Sample Go Hello World app for Shipt PoV" \
      org.opencontainers.image.url="https://github.com/parson-harness/sample-apps" \
      org.opencontainers.image.source="https://github.com/parson-harness/sample-apps" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT_SHA}" \
      org.opencontainers.image.created="${BUILD_TIME}" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.vendor="Harness"

WORKDIR /
COPY --from=build /out/app /app

ENV PORT=8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app"]
