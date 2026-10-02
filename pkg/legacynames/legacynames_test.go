package legacynames

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestLegacyName(t *testing.T) {
	for in, want := range map[string]string{
		"bylonis_latency":          "signoz_latency",
		"bylonis_calls_total":      "signoz_calls_total",
		"bylonis.meter.span.count": "signoz.meter.span.count",
	} {
		got, ok := LegacyName(in)
		assert.True(t, ok, in)
		assert.Equal(t, want, got)
	}
	for _, in := range []string{"bylonis", "bylonisx_latency", "signoz_latency", "http.server.duration", ""} {
		_, ok := LegacyName(in)
		assert.False(t, ok, in)
	}
}

func TestAppendLegacyCopies(t *testing.T) {
	ms := pmetric.NewMetricSlice()

	calls := ms.AppendEmpty()
	calls.SetName("bylonis_calls_total")
	dp := calls.SetEmptySum().DataPoints().AppendEmpty()
	dp.SetIntValue(3)
	dp.Attributes().PutStr("service.name", "frontend")
	dp.Attributes().PutStr(CollectorIDKey, "id-1")
	dp.Attributes().PutStr("resource_"+CollectorIDKey, "id-1")

	lat := ms.AppendEmpty()
	lat.SetName("bylonis_latency")
	lat.SetEmptyHistogram().DataPoints().AppendEmpty().Attributes().PutStr(CollectorIDKey, "id-1")

	ms.AppendEmpty().SetName("http.server.duration") // not ours: no copy

	AppendLegacyCopies(ms)

	require.Equal(t, 5, ms.Len())
	names := []string{}
	for i := 0; i < ms.Len(); i++ {
		names = append(names, ms.At(i).Name())
	}
	assert.Equal(t, []string{"bylonis_calls_total", "bylonis_latency", "http.server.duration", "signoz_calls_total", "signoz_latency"}, names)

	// The originals keep the bylonis attribute.
	orig := ms.At(0).Sum().DataPoints().At(0).Attributes()
	assert.Equal(t, map[string]any{"service.name": "frontend", CollectorIDKey: "id-1", "resource_" + CollectorIDKey: "id-1"}, orig.AsRaw())

	// The copies are identical but for the name and the legacy attribute.
	legacy := ms.At(3)
	assert.Equal(t, pmetric.MetricTypeSum, legacy.Type())
	assert.Equal(t, int64(3), legacy.Sum().DataPoints().At(0).IntValue())
	assert.Equal(t, map[string]any{"service.name": "frontend", LegacyCollectorIDKey: "id-1", "resource_" + LegacyCollectorIDKey: "id-1"}, legacy.Sum().DataPoints().At(0).Attributes().AsRaw())
	assert.Equal(t, map[string]any{LegacyCollectorIDKey: "id-1"}, ms.At(4).Histogram().DataPoints().At(0).Attributes().AsRaw())
}
