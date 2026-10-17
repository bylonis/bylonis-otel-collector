package schemamigrator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenameTable(t *testing.T) {
	op := RenameTableOperation{Database: "db", Table: "old", NewTable: "new"}
	require.Equal(t, "RENAME TABLE IF EXISTS db.old TO db.new", op.ToSQL())
	require.Equal(t, "RENAME TABLE IF EXISTS db.old TO db.new ON CLUSTER cluster", op.OnCluster("cluster").ToSQL())
	require.True(t, !op.IsMutation() && op.IsIdempotent() && op.IsLightweight(), "rename must run in sync mode")
}

func TestCreateTableAs(t *testing.T) {
	op := CreateTableAsOperation{
		Database: "db",
		Table:    "distributed_new",
		AsTable:  "distributed_old",
		Engine:   Distributed{Database: "db", Table: "new", ShardingKey: "cityHash64(id)"},
	}
	require.Equal(t, "CREATE TABLE IF NOT EXISTS db.distributed_new ON CLUSTER cluster AS db.distributed_old ENGINE = Distributed('cluster', db, new, cityHash64(id))", op.OnCluster("cluster").WithReplication().ToSQL())

	op.OrReplace = true
	require.Equal(t, "CREATE OR REPLACE TABLE db.distributed_new ON CLUSTER cluster AS db.distributed_old ENGINE = Distributed('cluster', db, new, cityHash64(id))", op.OnCluster("cluster").ToSQL())
	wait, _, _ := op.ShouldWaitForDistributionQueue()
	require.False(t, wait)
	require.True(t, !op.IsMutation() && op.IsIdempotent() && op.IsLightweight(), "create as must run in sync mode")
}

func TestMigration1011KeepsNoSignozSource(t *testing.T) {
	for _, op := range spanIndexV3MaterializedViews("bylonis_index_v3") {
		require.NotContains(t, op.ToSQL(), "signoz_index_v3")
		require.Contains(t, op.ToSQL(), "bylonis_traces.bylonis_index_v3")
	}
}

func TestMigration1013DropsAliasesAndUsageExplorer(t *testing.T) {
	var m SchemaMigrationRecord
	for _, r := range TracesMigrations {
		if r.MigrationID == 1013 {
			m = r
		}
	}
	require.Equal(t, uint64(1013), m.MigrationID)

	var up []string
	for _, op := range m.UpItems {
		up = append(up, op.OnCluster("cluster").ToSQL())
	}
	require.Equal(t, []string{
		"DROP TABLE IF EXISTS bylonis_traces.distributed_signoz_index_v3 ON CLUSTER cluster",
		"DROP TABLE IF EXISTS bylonis_traces.distributed_signoz_error_index_v2 ON CLUSTER cluster",
		"DROP TABLE IF EXISTS bylonis_traces.distributed_usage_explorer ON CLUSTER cluster",
		"DROP TABLE IF EXISTS bylonis_traces.usage_explorer ON CLUSTER cluster",
	}, up)

	// the down migration points the aliases at the renamed tables
	for _, op := range m.DownItems {
		require.Contains(t, op.ToSQL(), "AS bylonis_traces.distributed_bylonis_")
	}
}
