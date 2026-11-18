package schemamigrator

import (
	"fmt"
	"testing"
	"time"

	"github.com/bylonis/bylonis-otel-collector/constants"
	"github.com/stretchr/testify/require"
)

func TestInsertIntoTable(t *testing.T) {
	timestamp := time.Now().UnixMilli()

	testCases := []struct {
		name string
		op   Operation
		want string
	}{
		{
			name: "insert-into-table",
			op: InsertIntoTable{
				Database: constants.MetadataDB,
				Table:    "distributed_column_evolution_metadata",
				Columns:  []string{"signal", "column_name", "column_type", "field_context", "field_name", "version", "release_time"},
				Values:   fmt.Sprintf("('logs', 'body_promoted', 'JSON', 'body', 'message', 0, %d)", timestamp),
			},
			want: fmt.Sprintf("INSERT INTO %s.distributed_column_evolution_metadata (signal, column_name, column_type, field_context, field_name, version, release_time) VALUES ('logs', 'body_promoted', 'JSON', 'body', 'message', 0, %d)", constants.MetadataDB, timestamp),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.op.ToSQL())
		})
	}
}

func TestInsertIntoSelect(t *testing.T) {
	op := InsertIntoSelect{
		Database: "bylonis_metrics",
		Table:    "distributed_capacity_series",
		Columns:  []string{"env", "metric_name"},
		Query:    "SELECT env, metric_name FROM bylonis_metrics.distributed_time_series_v4",
	}
	require.Equal(t, "INSERT INTO bylonis_metrics.distributed_capacity_series (env, metric_name) SELECT env, metric_name FROM bylonis_metrics.distributed_time_series_v4", op.ToSQL())
	require.Equal(t, op.ToSQL(), op.OnCluster("cluster").ToSQL(), "INSERT has no ON CLUSTER")
	require.True(t, op.IsIdempotent())
	require.False(t, op.IsLightweight())
	require.False(t, op.IsMutation())
	require.False(t, op.ForceMigrate())
}
