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

### OpAMP token

With `--manager-config`, the collector gets its config from the OpAMP server of
the ByLonis backend (`server_endpoint`). Set `BYLONIS_OTEL_COLLECTOR_OPAMP_TOKEN`
(`--opamp-token`) and the collector sends `Authorization: Bearer <token>` when it
opens the WebSocket connection; the backend rejects connections without the
token it expects (`BYLONIS_OPAMP_TOKEN`). Without it, no header is sent and the
log says so with a warning.

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
