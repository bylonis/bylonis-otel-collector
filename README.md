# ByLonis OpenTelemetry Collector

The OpenTelemetry Collector distribution that ingests traces, logs and metrics
into ByLonis's ClickHouse, plus the schema migrator that creates and upgrades
those databases.

Hard fork of [SigNoz/signoz-otel-collector](https://github.com/SigNoz/signoz-otel-collector)
at `v0.144.5`, under the AGPL-3.0 (see [`LICENSE`](LICENSE) and [`NOTICE`](NOTICE)).

## Build

Requires Go 1.25.

```sh
make build        # .build/<os>-<arch>/bylonis-otel-collector and bylonis-schema-migrator
make test
```

## Run

```sh
bylonis-otel-collector migrate bootstrap   # create the databases
bylonis-otel-collector migrate sync up     # schema migrations
bylonis-otel-collector migrate async up    # background migrations
bylonis-otel-collector --config=config.yaml
```

## Configuration

Every flag can be set with an env var named `BYLONIS_OTEL_COLLECTOR_<FLAG>`
(`--clickhouse-dsn` → `BYLONIS_OTEL_COLLECTOR_CLICKHOUSE_DSN`).

| Env var | Default | Meaning |
|---|---|---|
| `BYLONIS_OTEL_COLLECTOR_CLICKHOUSE_DSN` | `tcp://0.0.0.0:9001` | ClickHouse used by `migrate` |
| `BYLONIS_OTEL_COLLECTOR_CLICKHOUSE_CLUSTER` | `cluster` | ClickHouse cluster name |
| `BYLONIS_OTEL_COLLECTOR_CLICKHOUSE_REPLICATION` | `true` | Replicated tables |
| `BYLONIS_DB_PREFIX` | `signoz` | Prefix of the database names: `<prefix>_traces`, `_metrics`, `_logs`, `_meter`, `_metadata`, `_analytics` |

`BYLONIS_DB_PREFIX` must be set to the same value for the migrator and the
collector. Apart from the database names, migrations, tables and queries don't
change.

### Logs pipelines preview

The ByLonis backend runs the logs pipelines preview, and validates pipelines
before saving them, in the collector of the tenant (`pkg/previewserver`): it
sends the pipeline processors it generates and sample logs, and gets the
processed logs back. The server starts with the collector, outside the config
managed over OpAMP, and only when a token is set.

| Env var | Default | Meaning |
|---|---|---|
| `BYLONIS_OTEL_COLLECTOR_PREVIEW_LISTEN` | `:4319` | Address of the preview server |
| `BYLONIS_OTEL_COLLECTOR_PREVIEW_TOKEN` | (none: off) | Bearer token, at least 16 characters. The backend uses the same one (`BYLONIS_LOGSPIPELINE_PREVIEW_TOKEN`) |

Keep the port on the internal network of the tenant: don't publish it. The
endpoint is `POST /v1/logs-pipelines/preview`; it only accepts
`bylonislogspipeline` processors, never expands `${...}` in their config and
never logs the logs it gets. Limits: 1 MiB body, 100 logs and 100 processors
per request, 10 s, 2 previews at once (429 beyond).

### Compatibility with SigNoz deployments

These keep working so an existing deployment can switch images without
changing anything else. They are deprecated and will be removed in a later
release.

- `SIGNOZ_OTEL_COLLECTOR_*` env vars are read when the `BYLONIS_` one is unset,
  with a warning in the log.
- Components keep their `signoz*` names (`signozlogspipeline`,
  `signozspanmetrics`, `signozclickhousemetrics`, `signozmeter`, ...). Each one is
  also available with `bylonis` in its place (`bylonislogspipeline`, ...).
- The images include `/signoz-otel-collector` and `/signoz-schema-migrator` as
  symlinks to the new binaries.
- With the default `BYLONIS_DB_PREFIX`, the databases are the upstream
  `signoz_*` ones.
