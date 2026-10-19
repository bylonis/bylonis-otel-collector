package components

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
)

func TestComponentsUseBylonisTypes(t *testing.T) {
	factories, err := Components()
	require.NoError(t, err)

	for _, tt := range []struct {
		kind  map[component.Type]bool
		types []string
	}{
		{keys(factories.Receivers), []string{"byloniskafkareceiver", "bylonisawsfirehose"}},
		{keys(factories.Processors), []string{"bylonisspanmetrics", "bylonis_tail_sampling", "bylonistransform", "bylonislogspipeline", "bylonisllmpricing"}},
		{keys(factories.Exporters), []string{"bylonisclickhousemeter", "bylonisclickhousemetrics", "byloniskafkaexporter", "bylonisjsontypeexporter"}},
		{keys(factories.Extensions), []string{"bylonis_health_check"}},
		{keys(factories.Connectors), []string{"bylonismeter"}},
	} {
		for _, typ := range tt.types {
			assert.True(t, tt.kind[component.MustNewType(typ)], "%s must be registered", typ)
		}
	}

	// The signoz* names are gone: a config that still uses one fails with an
	// unknown type error instead of running under the old name.
	for _, kind := range []map[component.Type]bool{
		keys(factories.Receivers), keys(factories.Processors), keys(factories.Exporters),
		keys(factories.Extensions), keys(factories.Connectors),
	} {
		for typ := range kind {
			assert.False(t, strings.HasPrefix(typ.String(), "signoz"), "%s is still registered", typ)
		}
	}
}

func keys[F any](m map[component.Type]F) map[component.Type]bool {
	out := make(map[component.Type]bool, len(m))
	for k := range m {
		out[k] = true
	}
	return out
}
