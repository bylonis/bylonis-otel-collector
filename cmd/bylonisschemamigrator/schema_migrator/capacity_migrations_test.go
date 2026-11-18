package schemamigrator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapacityMigrationsAppended(t *testing.T) {
	n := len(MetricsMigrations)
	require.GreaterOrEqual(t, n, 2)
	assert.Equal(t, uint64(1008), MetricsMigrations[n-2].MigrationID)
	assert.Equal(t, uint64(1009), MetricsMigrations[n-1].MigrationID)
}

func TestCapacityMigrationsNature(t *testing.T) {
	manager := newTestMigrationManager(t)
	// 1008 crea tablas y vistas: todo sync.
	assert.True(t, manager.IsSync(CapacityMigrations[0]))
	assert.False(t, manager.IsAsync(CapacityMigrations[0]))
	// 1009 copia datos: async, y la fase async solo ejecuta operaciones idempotentes.
	assert.True(t, manager.IsAsync(CapacityMigrations[1]))
	assert.False(t, manager.IsSync(CapacityMigrations[1]))
	for _, op := range CapacityMigrations[1].UpItems {
		assert.True(t, op.IsIdempotent())
	}
}

func TestCapacityDailyTable(t *testing.T) {
	sql := CapacityMigrations[0].UpItems[0].OnCluster("cluster").WithReplication().ToSQL()
	assert.Contains(t, sql, "CREATE TABLE IF NOT EXISTS bylonis_metrics.capacity_daily ON CLUSTER cluster")
	assert.Contains(t, sql, "last AggregateFunction(argMax, Float64, Int64)")
	assert.Contains(t, sql, "ReplicatedAggregatingMergeTree")
	assert.Contains(t, sql, "TTL toDateTime(unix_milli / 1000) + toIntervalSecond(34128000)")
	assert.Contains(t, sql, "PARTITION BY toYYYYMM(toDate(unix_milli / 1000))")
}

func TestCapacityViewsQualifyTime(t *testing.T) {
	daily := capacityDailyMVQuery()
	// Sin calificar, argMaxState usaría el alias unix_milli (el día).
	assert.Contains(t, daily, "argMaxState(toFloat64(last), samples_v4_agg_30m.unix_milli)")
	assert.Contains(t, daily, "FROM bylonis_metrics.samples_v4_agg_30m")
	assert.Contains(t, capacitySeriesMVQuery(), "intDiv(time_series_v4.unix_milli, 86400000)")
	for _, q := range []string{daily, capacitySeriesMVQuery(), capacityDailyBackfillQuery(), capacitySeriesBackfillQuery()} {
		assert.Contains(t, q, "'system.filesystem.usage'")
		assert.Contains(t, q, "'k8s.container.cpu_request'")
	}
}

func TestCapacityDailyBackfillGuard(t *testing.T) {
	q := capacityDailyBackfillQuery()
	assert.Contains(t, q, "FROM bylonis_metrics.distributed_samples_v4_agg_30m")
	assert.Contains(t, q, "AND unix_milli < toUnixTimestamp(toStartOfDay(now(), 'UTC')) * 1000")
	assert.Contains(t, q, "(SELECT count() FROM bylonis_metrics.distributed_capacity_daily WHERE unix_milli < toUnixTimestamp(toStartOfDay(now(), 'UTC')) * 1000) = 0")
	assert.True(t, strings.HasPrefix(CapacityMigrations[1].UpItems[1].ToSQL(), "INSERT INTO bylonis_metrics.distributed_capacity_daily (env, temporality, metric_name, fingerprint, unix_milli, last, min, max, sum, count) SELECT"))
}
