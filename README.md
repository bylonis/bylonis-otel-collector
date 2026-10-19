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

The databases are always `bylonis_traces`, `bylonis_metrics`, `bylonis_logs`,
`bylonis_meter`, `bylonis_metadata` and `bylonis_analytics`.

The components built for ByLonis are named `bylonis*`: `bylonisspanmetrics`,
`bylonismeter`, `bylonisclickhousemetrics`, `bylonisclickhousemeter`,
`bylonislogspipeline`, `bylonistransform`, `bylonis_tail_sampling`,
`bylonisllmpricing`, `byloniskafkareceiver`, `byloniskafkaexporter`,
`bylonisawsfirehose`, `bylonisjsontypeexporter` and `bylonis_health_check`.

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

### OpAMP token

With `--manager-config`, the collector gets its config from the OpAMP server of
the ByLonis backend (`server_endpoint`). Set `BYLONIS_OTEL_COLLECTOR_OPAMP_TOKEN`
(`--opamp-token`) and the collector sends `Authorization: Bearer <token>` when it
opens the WebSocket connection; the backend rejects connections without the
token it expects (`BYLONIS_OPAMP_TOKEN`). Without it, no header is sent and the
log says so with a warning.

### Upgrading from a SigNoz deployment or a version before v1.0

v1.0 drops the compatibility with the upstream names that earlier versions
kept. Before switching images, a deployment that still uses them has to:

| Before | Now | If it is left |
|---|---|---|
| `signoz*` component types (`signozspanmetrics`, `signozmeter`, `signoz_health_check`, ...) | `bylonis*` with the same suffix (`bylonisspanmetrics`, `bylonismeter`, `bylonis_health_check`, ...) | the config fails to load: `unknown type: "signozspanmetrics"` |
| `SIGNOZ_OTEL_COLLECTOR_<FLAG>` env vars | `BYLONIS_OTEL_COLLECTOR_<FLAG>` | the collector doesn't start and names the variable to rename |
| `BYLONIS_DB_PREFIX` (databases `signoz_*` or another prefix) | databases `bylonis_*`; unset the variable (`bylonis` is still accepted) | the collector and the migrator don't start |
| SQL naming `signoz_traces`, `signoz_logs`, ... (rewritten until now) | the `bylonis_*` names | ClickHouse answers that the database doesn't exist |
| `legacy_metric_names` in `bylonisspanmetrics` and `bylonismeter` (copies of the metrics as `signoz_*` and `signoz.meter.*`, with `signoz.collector.id`) | nothing: only the `bylonis_*` and `bylonis.meter.*` names, with `bylonis.collector.id` | `true` fails validation; `false` is ignored with a warning, and will be rejected in a later release |
| `/signoz-otel-collector` and `/signoz-schema-migrator` in the images | `/bylonis-otel-collector`, `/bylonis-schema-migrator` | the container doesn't start: `no such file or directory` |

The usage metering no longer skips metrics named `signoz*` or resource
attributes `signoz.workspace.*`; it skips `bylonis*` metrics and
`bylonis.workspace.*` attributes.
