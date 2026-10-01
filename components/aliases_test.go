package components

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
)

func TestComponentsRegisterBylonisAliases(t *testing.T) {
	factories, err := Components()
	require.NoError(t, err)

	// One per kind, plus the ones the backend pushes over OpAMP.
	for _, tt := range []struct {
		legacy, alias string
		kind          map[component.Type]bool
	}{
		{"signozlogspipeline", "bylonislogspipeline", keys(factories.Processors)},
		{"signozspanmetrics", "bylonisspanmetrics", keys(factories.Processors)},
		{"signoz_tail_sampling", "bylonis_tail_sampling", keys(factories.Processors)},
		{"signozclickhousemetrics", "bylonisclickhousemetrics", keys(factories.Exporters)},
		{"signozkafkareceiver", "byloniskafkareceiver", keys(factories.Receivers)},
		{"signoz_health_check", "bylonis_health_check", keys(factories.Extensions)},
		{"signozmeter", "bylonismeter", keys(factories.Connectors)},
	} {
		assert.True(t, tt.kind[component.MustNewType(tt.legacy)], "legacy %s must stay registered", tt.legacy)
		assert.True(t, tt.kind[component.MustNewType(tt.alias)], "alias %s must be registered", tt.alias)
	}

	// Every signoz* component has its alias, and the alias reports its own type
	// and the same default config as the original.
	for typ, f := range factories.Processors {
		if !strings.HasPrefix(typ.String(), "signoz") {
			continue
		}
		alias := factories.Processors[component.MustNewType("bylonis"+strings.TrimPrefix(typ.String(), "signoz"))]
		require.NotNil(t, alias, typ.String())
		assert.Equal(t, "bylonis"+strings.TrimPrefix(typ.String(), "signoz"), alias.Type().String())
		assert.Equal(t, f.CreateDefaultConfig(), alias.CreateDefaultConfig())
	}
}

func keys[F any](m map[component.Type]F) map[component.Type]bool {
	out := make(map[component.Type]bool, len(m))
	for k := range m {
		out[k] = true
	}
	return out
}
