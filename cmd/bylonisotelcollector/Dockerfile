# syntax=docker/dockerfile:1

# build on the runner's platform and cross-compile: the binary is pure Go (CGO off)
FROM --platform=$BUILDPLATFORM golang:1.25-bookworm AS build

ARG TARGETOS TARGETARCH
ARG VERSION=dev

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath \
    -ldflags "-s -w -X github.com/bylonis/bylonis-otel-collector/constants.Version=${VERSION}" \
    -o /out/bylonis-otel-collector ./cmd/bylonisotelcollector

# use a minimal debian image
FROM debian:bookworm-slim

# add ca-certificates in case you need them
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# transitional alias: existing deployments run /signoz-otel-collector in their compose
# commands. Remove once every deployment calls /bylonis-otel-collector.
RUN ln -s /bylonis-otel-collector /signoz-otel-collector

# define arguments and default values
ARG USER_UID=10001

# create a non-root user for running the collector
USER ${USER_UID}

# copy the binary from the build stage
COPY --from=build /out/bylonis-otel-collector /bylonis-otel-collector

# copy the config file
COPY conf/default.yaml /etc/otel/config.yaml

# expose OTLP ports for the collector
EXPOSE 4317 4318

# run the binary as the entrypoint and pass the default config file as a flag
ENTRYPOINT [ "/bylonis-otel-collector" ]
CMD ["--config", "/etc/otel/config.yaml"]
