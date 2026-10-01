package dbprefix

import (
	"context"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	for _, tt := range []struct {
		name    string
		env     map[string]string
		want    string
		wantErr bool
	}{
		{name: "unset", want: "signoz"},
		{name: "empty", env: map[string]string{EnvVar: ""}, want: "signoz"},
		{name: "bylonis", env: map[string]string{EnvVar: "bylonis"}, want: "bylonis"},
		{name: "uppercase", env: map[string]string{EnvVar: "ByLonis"}, wantErr: true},
		{name: "sql injection", env: map[string]string{EnvVar: "x; DROP"}, wantErr: true},
		{name: "starts with digit", env: map[string]string{EnvVar: "1abc"}, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := load(func(k string) (string, bool) { v, ok := tt.env[k]; return v, ok })
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDefaultNames(t *testing.T) {
	// The test binary runs without BYLONIS_DB_PREFIX: upstream names.
	require.True(t, IsDefault())
	assert.Equal(t, "signoz_traces", Traces())
	assert.Equal(t, "signoz_metrics", Metrics())
	assert.Equal(t, "signoz_logs", Logs())
	assert.Equal(t, "signoz_meter", Meter())
	assert.Equal(t, "signoz_metadata", Metadata())
	assert.Equal(t, "signoz_analytics", Analytics())
	assert.Equal(t, "SELECT * FROM signoz_logs.logs_v2", Rewrite("SELECT * FROM signoz_logs.logs_v2"))
}

func TestRewriteWith(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"CREATE DATABASE IF NOT EXISTS signoz_traces ON CLUSTER cluster", "CREATE DATABASE IF NOT EXISTS bylonis_traces ON CLUSTER cluster"},
		{"SELECT * FROM signoz_traces.signoz_index_v3", "SELECT * FROM bylonis_traces.signoz_index_v3"},
		{"Distributed('cluster', 'signoz_logs', 'logs_v2', cityHash64(id))", "Distributed('cluster', 'bylonis_logs', 'logs_v2', cityHash64(id))"},
		{"TO signoz_metrics.samples_v4_agg_5m AS SELECT * FROM signoz_metrics.samples_v4", "TO bylonis_metrics.samples_v4_agg_5m AS SELECT * FROM bylonis_metrics.samples_v4"},
		{"signoz_meter signoz_metadata signoz_analytics", "bylonis_meter bylonis_metadata bylonis_analytics"},
		// not database names: left alone
		{"signoz_logs_v2 signoz_index_v3 signoz_error_index_v2 xsignoz_logs", "signoz_logs_v2 signoz_index_v3 signoz_error_index_v2 xsignoz_logs"},
		{"no names here", "no names here"},
	} {
		assert.Equal(t, tt.want, rewriteWith("bylonis", tt.in), tt.in)
		assert.Equal(t, tt.in, rewriteWith(Default, tt.in), "default prefix must not rewrite")
	}
}

type recordingConn struct {
	driver.Conn
	query string
	args  []any
}

func (c *recordingConn) Exec(_ context.Context, query string, args ...any) error {
	c.query, c.args = query, args
	return nil
}

func TestRewritingConn(t *testing.T) {
	rec := &recordingConn{}
	conn := &rewritingConn{Conn: rec, prefix: "bylonis"}
	args := []any{"cluster", "signoz_logs", 3, "signoz_logs_v2"}

	require.NoError(t, conn.Exec(context.Background(), "SELECT count() FROM system.tables WHERE database = $2 AND name = 'x' -- signoz_logs", args...))

	assert.Equal(t, "SELECT count() FROM system.tables WHERE database = $2 AND name = 'x' -- bylonis_logs", rec.query)
	assert.Equal(t, []any{"cluster", "bylonis_logs", 3, "signoz_logs_v2"}, rec.args)
	assert.Equal(t, "signoz_logs", args[1], "caller's args must not be mutated")
}
