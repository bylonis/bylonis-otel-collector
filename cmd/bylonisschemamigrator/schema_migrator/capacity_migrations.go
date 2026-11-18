package schemamigrator

import (
	"fmt"
	"strings"
)

// Resumen diario de métricas de capacidad (bylonis#887, ADR-001 D6 de Capacity
// Planning en el repo bylonis: docs/capacity-planning/adr-001-predict.md).
//
// Las métricas crudas y sus etiquetas (time_series_v4*) se guardan 30 días. Para
// ver tendencias de meses, dos tablas guardan solo las métricas del catálogo de
// capacidad, con su propia retención (395 días por defecto, configurable desde
// Ajustes → General → Retención):
//   - capacity_daily: por serie y día, máximo, mínimo, suma, cuenta y último valor.
//   - capacity_series: las etiquetas de cada serie, porque las de time_series_v4
//     caducan antes.
//
// La migración 1008 crea las tablas y las vistas materializadas (fase sync); la
// 1009 copia los días que ya hay (fase async).

// CapacityMetricNames son las métricas del catálogo de capacidad (ADR-001 D7).
// Todas son gauges o sumas no monótonas, así que su media diaria tiene sentido.
// Para añadir una, hace falta una migración nueva que cambie las dos vistas.
var CapacityMetricNames = []string{
	"system.filesystem.usage",
	"system.filesystem.utilization",
	"system.memory.usage",
	"system.memory.utilization",
	"k8s.node.filesystem.usage",
	"k8s.node.filesystem.capacity",
	"k8s.node.memory.working_set",
	"k8s.node.allocatable_memory",
	"k8s.node.cpu.usage",
	"k8s.node.allocatable_cpu",
	"k8s.pod.memory.working_set",
	"k8s.container.memory_limit",
	"k8s.pod.cpu.usage",
	"k8s.container.cpu_request",
	"k8s.volume.available",
	"k8s.volume.capacity",
}

// CapacityDefaultTTLSeconds es la retención por defecto del resumen: 395 días.
const CapacityDefaultTTLSeconds = 395 * 24 * 3600

const capacityDayMillis = 86400000

func capacityMetricsIn() string {
	quoted := make([]string, len(CapacityMetricNames))
	for i, name := range CapacityMetricNames {
		quoted[i] = "'" + name + "'"
	}
	return "metric_name IN (" + strings.Join(quoted, ", ") + ")"
}

func capacityDailyColumns() []Column {
	return []Column{
		{Name: "env", Type: LowCardinalityColumnType{ColumnTypeString}, Default: "'default'"},
		{Name: "temporality", Type: LowCardinalityColumnType{ColumnTypeString}, Default: "'Unspecified'"},
		{Name: "metric_name", Type: LowCardinalityColumnType{ColumnTypeString}},
		{Name: "fingerprint", Type: ColumnTypeUInt64, Codec: "ZSTD(1)"},
		{Name: "unix_milli", Type: ColumnTypeInt64, Codec: "Delta(8), ZSTD(1)"},
		// Último valor del día: argMax por el bucket de 30 min, porque las filas
		// parciales del día se funden en cualquier orden. Es un valor de la última
		// media hora con datos: samples_v4_agg_30m guarda last con anyLast.
		// min, max, sum y count son exactos.
		{Name: "last", Type: AggregateFunction{FunctionName: "argMax", Arguments: []ColumnType{ColumnTypeFloat64, ColumnTypeInt64}}, Codec: "ZSTD(1)"},
		{Name: "min", Type: SimpleAggregateFunction{FunctionName: "min", Arguments: []ColumnType{ColumnTypeFloat64}}, Codec: "ZSTD(1)"},
		{Name: "max", Type: SimpleAggregateFunction{FunctionName: "max", Arguments: []ColumnType{ColumnTypeFloat64}}, Codec: "ZSTD(1)"},
		{Name: "sum", Type: SimpleAggregateFunction{FunctionName: "sum", Arguments: []ColumnType{ColumnTypeFloat64}}, Codec: "ZSTD(1)"},
		{Name: "count", Type: SimpleAggregateFunction{FunctionName: "sum", Arguments: []ColumnType{ColumnTypeUInt64}}, Codec: "ZSTD(1)"},
	}
}

func capacitySeriesColumns() []Column {
	return []Column{
		{Name: "env", Type: LowCardinalityColumnType{ColumnTypeString}, Default: "'default'"},
		{Name: "temporality", Type: LowCardinalityColumnType{ColumnTypeString}, Default: "'Unspecified'"},
		{Name: "metric_name", Type: LowCardinalityColumnType{ColumnTypeString}},
		{Name: "fingerprint", Type: ColumnTypeUInt64, Codec: "ZSTD(1)"},
		// Último día en que se vio la serie: la fila caduca a partir de él.
		{Name: "unix_milli", Type: ColumnTypeInt64, Codec: "Delta(8), ZSTD(1)"},
		{Name: "labels", Type: ColumnTypeString, Codec: "ZSTD(5)"},
	}
}

func capacityTTL() string {
	return fmt.Sprintf("toDateTime(unix_milli / 1000) + toIntervalSecond(%d)", CapacityDefaultTTLSeconds)
}

// La columna de tiempo va calificada con la tabla: sin calificar, ClickHouse
// usaría el alias unix_milli (el día) dentro de argMaxState.
func capacityDailyMVQuery() string {
	return fmt.Sprintf(`SELECT
							env,
							temporality,
							metric_name,
							fingerprint,
							intDiv(samples_v4_agg_30m.unix_milli, %[1]d) * %[1]d AS unix_milli,
							argMaxState(toFloat64(last), samples_v4_agg_30m.unix_milli) AS last,
							min(min) AS min,
							max(max) AS max,
							sum(sum) AS sum,
							sum(count) AS count
						FROM bylonis_metrics.samples_v4_agg_30m
						WHERE %[2]s
						GROUP BY
							env,
							temporality,
							metric_name,
							fingerprint,
							unix_milli`, capacityDayMillis, capacityMetricsIn())
}

func capacitySeriesMVQuery() string {
	return fmt.Sprintf(`SELECT
							env,
							temporality,
							metric_name,
							fingerprint,
							intDiv(time_series_v4.unix_milli, %[1]d) * %[1]d AS unix_milli,
							labels
						FROM bylonis_metrics.time_series_v4
						WHERE %[2]s`, capacityDayMillis, capacityMetricsIn())
}

// startOfTodayMillis es el comienzo del día UTC actual en milisegundos.
const startOfTodayMillis = "toUnixTimestamp(toStartOfDay(now(), 'UTC')) * 1000"

// capacityDailyBackfillQuery copia los días completos anteriores a hoy. Hoy lo
// rellena ya la vista materializada. La guarda final hace que repetirla no
// duplique nada: si ya hay algún día anterior a hoy, no inserta.
func capacityDailyBackfillQuery() string {
	return fmt.Sprintf(`SELECT
							env,
							temporality,
							metric_name,
							fingerprint,
							intDiv(unix_milli, %[1]d) * %[1]d AS day,
							argMaxState(toFloat64(last), unix_milli),
							min(min),
							max(max),
							sum(sum),
							sum(count)
						FROM bylonis_metrics.distributed_samples_v4_agg_30m
						WHERE %[2]s
							AND unix_milli < %[3]s
							AND (SELECT count() FROM bylonis_metrics.distributed_capacity_daily WHERE unix_milli < %[3]s) = 0
						GROUP BY
							env,
							temporality,
							metric_name,
							fingerprint,
							day`, capacityDayMillis, capacityMetricsIn(), startOfTodayMillis)
}

// capacitySeriesBackfillQuery copia las etiquetas de las series vivas. Repetirla
// es inocuo: capacity_series es un ReplacingMergeTree.
func capacitySeriesBackfillQuery() string {
	return fmt.Sprintf(`SELECT
							env,
							temporality,
							metric_name,
							fingerprint,
							intDiv(max(unix_milli), %[1]d) * %[1]d,
							argMax(labels, unix_milli)
						FROM bylonis_metrics.distributed_time_series_v4
						WHERE %[2]s
						GROUP BY
							env,
							temporality,
							metric_name,
							fingerprint`, capacityDayMillis, capacityMetricsIn())
}

var capacityShardingKey = "cityHash64(env, temporality, metric_name, fingerprint)"

// CapacityMigrations se añaden al final de MetricsMigrations.
var CapacityMigrations = []SchemaMigrationRecord{
	{
		MigrationID: 1008,
		UpItems: []Operation{
			CreateTableOperation{
				Database: "bylonis_metrics",
				Table:    "capacity_daily",
				Columns:  capacityDailyColumns(),
				Engine: AggregatingMergeTree{
					MergeTree{
						PartitionBy: "toYYYYMM(toDate(unix_milli / 1000))",
						OrderBy:     "(env, temporality, metric_name, fingerprint, unix_milli)",
						TTL:         capacityTTL(),
						Settings: TableSettings{
							{Name: "index_granularity", Value: "8192"},
							{Name: "ttl_only_drop_parts", Value: "1"},
						},
					},
				},
			},
			CreateTableOperation{
				Database: "bylonis_metrics",
				Table:    "distributed_capacity_daily",
				Columns:  capacityDailyColumns(),
				Engine: Distributed{
					Database:    "bylonis_metrics",
					Table:       "capacity_daily",
					ShardingKey: capacityShardingKey,
				},
			},
			CreateTableOperation{
				Database: "bylonis_metrics",
				Table:    "capacity_series",
				Columns:  capacitySeriesColumns(),
				Engine: ReplacingMergeTree{
					MergeTree{
						OrderBy: "(env, temporality, metric_name, fingerprint)",
						TTL:     capacityTTL(),
						Settings: TableSettings{
							{Name: "index_granularity", Value: "8192"},
						},
					},
				},
			},
			CreateTableOperation{
				Database: "bylonis_metrics",
				Table:    "distributed_capacity_series",
				Columns:  capacitySeriesColumns(),
				Engine: Distributed{
					Database:    "bylonis_metrics",
					Table:       "capacity_series",
					ShardingKey: capacityShardingKey,
				},
			},
			CreateMaterializedViewOperation{
				Database:  "bylonis_metrics",
				ViewName:  "capacity_daily_mv",
				DestTable: "capacity_daily",
				Query:     capacityDailyMVQuery(),
			},
			CreateMaterializedViewOperation{
				Database:  "bylonis_metrics",
				ViewName:  "capacity_series_mv",
				DestTable: "capacity_series",
				Query:     capacitySeriesMVQuery(),
			},
		},
		DownItems: []Operation{
			DropTableOperation{Database: "bylonis_metrics", Table: "capacity_series_mv"},
			DropTableOperation{Database: "bylonis_metrics", Table: "capacity_daily_mv"},
			DropTableOperation{Database: "bylonis_metrics", Table: "distributed_capacity_series"},
			DropTableOperation{Database: "bylonis_metrics", Table: "capacity_series"},
			DropTableOperation{Database: "bylonis_metrics", Table: "distributed_capacity_daily"},
			DropTableOperation{Database: "bylonis_metrics", Table: "capacity_daily"},
		},
	},
	{
		MigrationID: 1009,
		UpItems: []Operation{
			InsertIntoSelect{
				Database: "bylonis_metrics",
				Table:    "distributed_capacity_series",
				Columns:  []string{"env", "temporality", "metric_name", "fingerprint", "unix_milli", "labels"},
				Query:    capacitySeriesBackfillQuery(),
			},
			InsertIntoSelect{
				Database: "bylonis_metrics",
				Table:    "distributed_capacity_daily",
				Columns:  []string{"env", "temporality", "metric_name", "fingerprint", "unix_milli", "last", "min", "max", "sum", "count"},
				Query:    capacityDailyBackfillQuery(),
			},
		},
	},
}

func init() {
	MetricsMigrations = append(MetricsMigrations, CapacityMigrations...)
}
