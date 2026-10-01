# syntax=docker/dockerfile:1

# build on the runner's platform and cross-compile: the binary is pure Go (CGO off)
FROM --platform=$BUILDPLATFORM golang:1.25-bookworm AS build

ARG TARGETOS TARGETARCH

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags "-s -w" \
    -o /out/bylonis-schema-migrator ./cmd/bylonisschemamigrator

# use a minimal debian image
FROM debian:bookworm-slim

# define arguments and default values
ARG USER_UID=10001

# transitional alias for deployments that still call /signoz-schema-migrator
RUN ln -s /bylonis-schema-migrator /signoz-schema-migrator

# create a non-root user for running the migrator
USER ${USER_UID}

# copy the binary from the build stage
COPY --from=build /out/bylonis-schema-migrator /bylonis-schema-migrator

# run the binary as the entrypoint and pass the default dsn as a flag
ENTRYPOINT [ "/bylonis-schema-migrator" ]
CMD ["--dsn", "tcp://localhost:9000"]
