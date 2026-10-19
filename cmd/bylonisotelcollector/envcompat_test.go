package main

import (
	"testing"

	"github.com/bylonis/bylonis-otel-collector/cmd/bylonisotelcollector/config"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func newTestFlags() *flag.FlagSet {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	flags.String("clickhouse-dsn", "tcp://0.0.0.0:9001", "")
	flags.Bool("clickhouse-replication", true, "")
	return flags
}

func lookupFrom(env map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	}
}

func TestSetFlagsFromEnv(t *testing.T) {
	tests := []struct {
		name       string
		env        map[string]string
		args       []string
		wantDSN    string
		wantRepl   string
		wantWarned bool
	}{
		{
			name:     "defaults when nothing is set",
			wantDSN:  "tcp://0.0.0.0:9001",
			wantRepl: "true",
		},
		{
			name:     "bylonis variable",
			env:      map[string]string{"BYLONIS_OTEL_COLLECTOR_CLICKHOUSE_DSN": "tcp://new:9000"},
			wantDSN:  "tcp://new:9000",
			wantRepl: "true",
		},
		{
			name:       "legacy signoz variable with warning",
			env:        map[string]string{"SIGNOZ_OTEL_COLLECTOR_CLICKHOUSE_REPLICATION": "false"},
			wantDSN:    "tcp://0.0.0.0:9001",
			wantRepl:   "false",
			wantWarned: true,
		},
		{
			name: "bylonis wins over signoz",
			env: map[string]string{
				"BYLONIS_OTEL_COLLECTOR_CLICKHOUSE_DSN": "tcp://new:9000",
				"SIGNOZ_OTEL_COLLECTOR_CLICKHOUSE_DSN":  "tcp://old:9000",
			},
			wantDSN:  "tcp://new:9000",
			wantRepl: "true",
		},
		{
			name: "empty bylonis variable falls back to signoz",
			env: map[string]string{
				"BYLONIS_OTEL_COLLECTOR_CLICKHOUSE_DSN": "",
				"SIGNOZ_OTEL_COLLECTOR_CLICKHOUSE_DSN":  "tcp://old:9000",
			},
			wantDSN:    "tcp://old:9000",
			wantRepl:   "true",
			wantWarned: true,
		},
		{
			name:     "command line wins over env",
			env:      map[string]string{"BYLONIS_OTEL_COLLECTOR_CLICKHOUSE_DSN": "tcp://new:9000"},
			args:     []string{"--clickhouse-dsn=tcp://flag:9000"},
			wantDSN:  "tcp://flag:9000",
			wantRepl: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := newTestFlags()
			require.NoError(t, flags.Parse(tt.args))
			core, logs := observer.New(zap.WarnLevel)

			require.NoError(t, setFlagsFromEnv(flags, lookupFrom(tt.env), zap.New(core)))

			assert.Equal(t, tt.wantDSN, flags.Lookup("clickhouse-dsn").Value.String())
			assert.Equal(t, tt.wantRepl, flags.Lookup("clickhouse-replication").Value.String())
			assert.Equal(t, tt.wantWarned, logs.Len() > 0)
		})
	}
}

func TestSetFlagsFromEnvInvalidValue(t *testing.T) {
	flags := newTestFlags()
	require.NoError(t, flags.Parse(nil))

	err := setFlagsFromEnv(flags, lookupFrom(map[string]string{"BYLONIS_OTEL_COLLECTOR_CLICKHOUSE_REPLICATION": "maybe"}), zap.NewNop())

	assert.ErrorContains(t, err, "BYLONIS_OTEL_COLLECTOR_CLICKHOUSE_REPLICATION")
}

// The OpAMP token of the collector command comes from BYLONIS_OTEL_COLLECTOR_OPAMP_TOKEN.
func TestSetFlagsFromEnvOpAMPToken(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	config.Collector.RegisterFlags(cmd)
	t.Cleanup(func() { config.Collector.OpAMPToken = "" })
	flags := cmd.PersistentFlags()
	require.NoError(t, flags.Parse(nil))

	env := map[string]string{"BYLONIS_OTEL_COLLECTOR_OPAMP_TOKEN": "s3cret-token"}
	require.NoError(t, setFlagsFromEnv(flags, lookupFrom(env), zap.NewNop()))

	assert.Equal(t, "s3cret-token", config.Collector.OpAMPToken)
}
