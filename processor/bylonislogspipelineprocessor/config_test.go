// Brought in as is from logstransform processor in opentelemetry-collector-contrib
package bylonislogspipelineprocessor

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap/confmaptest"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/entry"

	bylonislogspipelinestanzaadapter "github.com/bylonis/bylonis-otel-collector/processor/bylonislogspipelineprocessor/stanza/adapter"
	bylonisstanzaentry "github.com/bylonis/bylonis-otel-collector/processor/bylonislogspipelineprocessor/stanza/entry"
	bylonislogspipelinestanzaoperator "github.com/bylonis/bylonis-otel-collector/processor/bylonislogspipelineprocessor/stanza/operator"
	bylonisstanzahelper "github.com/bylonis/bylonis-otel-collector/processor/bylonislogspipelineprocessor/stanza/operator/helper"
	"github.com/bylonis/bylonis-otel-collector/processor/bylonislogspipelineprocessor/stanza/operator/operators/regex"
)

func TestLoadConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, cm.Unmarshal(cfg))
	assert.Equal(t, &Config{
		BaseConfig: bylonislogspipelinestanzaadapter.BaseConfig{
			Operators: []bylonislogspipelinestanzaoperator.Config{
				{
					Builder: func() *regex.Config {
						cfg := regex.NewConfig()
						cfg.Regex = "^(?P<time>\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2}) (?P<sev>[A-Z]*) (?P<msg>.*)$"
						sevField := bylonisstanzaentry.Field{FieldInterface: entry.NewAttributeField("sev")}
						sevCfg := bylonisstanzahelper.NewSeverityConfig()
						sevCfg.ParseFrom = &sevField
						cfg.SeverityConfig = &sevCfg
						timeField := bylonisstanzaentry.Field{FieldInterface: entry.NewAttributeField("time")}
						timeCfg := bylonisstanzahelper.NewTimeParser()
						timeCfg.Layout = "%Y-%m-%d %H:%M:%S"
						timeCfg.ParseFrom = &timeField
						cfg.TimeParser = &timeCfg
						return cfg
					}(),
				},
			},
		},
	}, cfg)
}
