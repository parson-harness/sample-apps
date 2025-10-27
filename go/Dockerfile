# --- build stage ---
ARG REGISTRY_OVERRIDE=docker.io
ARG GO_VERSION=1.25.3
FROM ${REGISTRY_OVERRIDE}/golang:${GO_VERSION}-alpine AS build

# Ensure auto toolchain stays enabled if you later bump 'go' in go.mod
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOTOOLCHAIN=auto

WORKDIR /src
COPY go.mod go.sum ./
# (If using BuildKit, these cache mounts help; safe to keep even if ignored)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /out/app \
      -ldflags "-s -w -X 'main.version=${APP_VERSION:-dev}' -X 'main.env=${APP_ENV:-dev}' -X 'main.buildTime=${BUILD_TIME:-unknown}'" \
      .

# --- runtime stage ---
FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=build /out/app /app
ENV PORT=8080
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/app"]
