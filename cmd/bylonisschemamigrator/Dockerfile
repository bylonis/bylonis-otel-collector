# use a minimal debian image
FROM debian:bookworm-slim

# define arguments and default values
ARG TARGETOS TARGETARCH
ARG USER_UID=10001

# transitional alias for deployments that still call /signoz-schema-migrator
RUN ln -s /bylonis-schema-migrator /signoz-schema-migrator

# create a non-root user for running the migrator
USER ${USER_UID}

# copy the binaries from the multi-stage build
COPY .build/${TARGETOS}-${TARGETARCH}/bylonis-schema-migrator /bylonis-schema-migrator

# run the binary as the entrypoint and pass the default dsn as a flag
ENTRYPOINT [ "/bylonis-schema-migrator" ]
CMD ["--dsn", "tcp://localhost:9000"]
