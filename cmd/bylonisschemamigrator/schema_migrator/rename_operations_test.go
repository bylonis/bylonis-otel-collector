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
