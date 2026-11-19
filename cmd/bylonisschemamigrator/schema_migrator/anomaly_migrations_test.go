package schemamigrator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnomalyMigrationsAppended(t *testing.T) {
	n := len(AnalyticsMigrations)
	require.GreaterOrEqual(t, n, 2)
	assert.Equal(t, uint64(1), AnalyticsMigrations[0].MigrationID)
	assert.Equal(t, uint64(2), AnalyticsMigrations[n-1].MigrationID)
}

func TestAnomalyMigrationsAreSync(t *testing.T) {
	manager := newTestMigrationManager(t)
	assert.True(t, manager.IsSync(AnomalyMigrations[0]))
	assert.False(t, manager.IsAsync(AnomalyMigrations[0]))
}

func TestAnomalyEvaluationsTable(t *testing.T) {
	sql := AnomalyMigrations[0].UpItems[0].OnCluster("cluster").WithReplication().ToSQL()
	assert.Contains(t, sql, "CREATE TABLE IF NOT EXISTS bylonis_analytics.anomaly_evaluations_v0 ON CLUSTER cluster")
	assert.Contains(t, sql, "ReplicatedReplacingMergeTree")
	assert.Contains(t, sql, "ORDER BY (rule_id, fingerprint, unix_milli)")
	assert.Contains(t, sql, "TTL toDateTime(unix_milli / 1000) + toIntervalDay(_retention_days)")
	assert.Contains(t, sql, "_retention_days UInt32 DEFAULT 30")
	for _, col := range []string{"expected Float64", "sigma Float64", "z Float64", "learning Bool", "masked Bool", "seasonality LowCardinality(String)"} {
		assert.Contains(t, sql, col)
	}

	dist := AnomalyMigrations[0].UpItems[1].OnCluster("cluster").ToSQL()
	assert.Contains(t, dist, "bylonis_analytics.distributed_anomaly_evaluations_v0")
	assert.Contains(t, dist, "Distributed('cluster', bylonis_analytics, anomaly_evaluations_v0, cityHash64(rule_id, fingerprint))")
	assert.NotContains(t, dist, "_retention_days")
}
