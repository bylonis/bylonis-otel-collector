package schemamigrator

// Evaluaciones de las reglas de anomalía (bylonis#900, ADR-001 D8 de Anomaly
// Detection en el repo bylonis: docs/anomaly-detection/adr-001-anomaly-rule.md).
//
// En cada evaluación, el gestor de reglas guarda los buckets cerrados nuevos de cada
// serie con su esperado, σ y z. Así la vista de una alerta pasada dibuja la banda de
// entonces aunque el modelo se reajuste después, y la distribución de z sirve para
// sugerir el umbral (bylonis#902).
//
// Es un ReplacingMergeTree por (rule_id, fingerprint, unix_milli): si el servidor se
// reinicia y vuelve a guardar un bucket, la fila se reemplaza en vez de duplicarse.
// Retención: 30 días (_retention_days).

var anomalyEvaluationsShardingKey = "cityHash64(rule_id, fingerprint)"

func anomalyEvaluationsColumns(withRetention bool) []Column {
	cols := []Column{}
	if withRetention {
		cols = append(cols, Column{Name: "_retention_days", Type: ColumnTypeUInt32, Default: "30"})
	}
	return append(cols,
		Column{Name: "rule_id", Type: LowCardinalityColumnType{ElementType: ColumnTypeString}},
		Column{Name: "fingerprint", Type: ColumnTypeUInt64, Codec: "ZSTD(1)"},
		Column{Name: "labels", Type: ColumnTypeString, Codec: "ZSTD(5)"},
		Column{Name: "unix_milli", Type: ColumnTypeInt64, Codec: "Delta(8), ZSTD(1)"},
		Column{Name: "value", Type: ColumnTypeFloat64, Codec: "Gorilla, ZSTD(1)"},
		Column{Name: "expected", Type: ColumnTypeFloat64, Codec: "Gorilla, ZSTD(1)"},
		Column{Name: "sigma", Type: ColumnTypeFloat64, Codec: "Gorilla, ZSTD(1)"},
		Column{Name: "z", Type: ColumnTypeFloat64, Codec: "Gorilla, ZSTD(1)"},
		Column{Name: "seasonality", Type: LowCardinalityColumnType{ElementType: ColumnTypeString}},
		Column{Name: "model", Type: LowCardinalityColumnType{ElementType: ColumnTypeString}},
		Column{Name: "learning", Type: ColumnTypeBool},
		Column{Name: "masked", Type: ColumnTypeBool},
	)
}

// AnomalyMigrations se añaden al final de AnalyticsMigrations.
var AnomalyMigrations = []SchemaMigrationRecord{
	{
		MigrationID: 2,
		UpItems: []Operation{
			CreateTableOperation{
				Database: "bylonis_analytics",
				Table:    "anomaly_evaluations_v0",
				Columns:  anomalyEvaluationsColumns(true),
				Engine: ReplacingMergeTree{
					MergeTree{
						PartitionBy: "toDate(unix_milli / 1000)",
						OrderBy:     "(rule_id, fingerprint, unix_milli)",
						TTL:         "toDateTime(unix_milli / 1000) + toIntervalDay(_retention_days)",
						Settings: TableSettings{
							{Name: "ttl_only_drop_parts", Value: "1"},
							{Name: "index_granularity", Value: "8192"},
						},
					},
				},
			},
			CreateTableOperation{
				Database: "bylonis_analytics",
				Table:    "distributed_anomaly_evaluations_v0",
				Columns:  anomalyEvaluationsColumns(false),
				Engine: Distributed{
					Database:    "bylonis_analytics",
					Table:       "anomaly_evaluations_v0",
					ShardingKey: anomalyEvaluationsShardingKey,
				},
			},
		},
		DownItems: []Operation{
			DropTableOperation{Database: "bylonis_analytics", Table: "distributed_anomaly_evaluations_v0"},
			DropTableOperation{Database: "bylonis_analytics", Table: "anomaly_evaluations_v0"},
		},
	},
}

func init() {
	AnalyticsMigrations = append(AnalyticsMigrations, AnomalyMigrations...)
}
