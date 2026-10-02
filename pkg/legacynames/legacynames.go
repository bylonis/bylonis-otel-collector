// Package legacynames emits the metrics the collector generates under their
// legacy SigNoz names as well as the bylonis ones, so that dashboards, alerts
// and UIs that still query signoz_* keep working while they move to the new
// names (bylonis/bylonis#47).
//
// The metric name is stored in every sample, so the two names are two
// independent sets of series. A legacy copy is identical to the original except
// for the metric name and the collector id attribute, which go back to their
// upstream form: the legacy series keep exactly the labels they had before.
package legacynames

import (
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

const (
	// Prefix and LegacyPrefix head the metric names: bylonis_latency and
	// signoz_latency, bylonis.meter.span.count and signoz.meter.span.count.
	Prefix       = "bylonis"
	LegacyPrefix = "signoz"

	// CollectorIDKey is the attribute spanmetrics sets to the collector instance.
	CollectorIDKey       = "bylonis.collector.id"
	LegacyCollectorIDKey = "signoz.collector.id"
)

// LegacyName returns the legacy name of a bylonis metric name ("bylonis_x" or
// "bylonis.x"), and false for names that don't have one.
func LegacyName(name string) (string, bool) {
	rest, ok := strings.CutPrefix(name, Prefix)
	if !ok || rest == "" || (rest[0] != '_' && rest[0] != '.') {
		return "", false
	}
	return LegacyPrefix + rest, true
}

// legacyAttr returns the legacy form of an attribute key: the collector id,
// bare or with a prefix such as "resource_".
func legacyAttr(key string) (string, bool) {
	if p, ok := strings.CutSuffix(key, CollectorIDKey); ok {
		return p + LegacyCollectorIDKey, true
	}
	return "", false
}

// AppendLegacyCopies appends to ms a legacy copy of every metric in ms that
// has a bylonis name.
func AppendLegacyCopies(ms pmetric.MetricSlice) {
	n := ms.Len()
	for i := 0; i < n; i++ {
		legacy, ok := LegacyName(ms.At(i).Name())
		if !ok {
			continue
		}
		c := ms.AppendEmpty()
		ms.At(i).CopyTo(c)
		c.SetName(legacy)
		forEachAttributes(c, renameAttributes)
	}
}

func renameAttributes(attrs pcommon.Map) {
	var keys []string
	attrs.Range(func(k string, _ pcommon.Value) bool {
		if _, ok := legacyAttr(k); ok {
			keys = append(keys, k)
		}
		return true
	})
	for _, k := range keys {
		v, _ := attrs.Get(k)
		nk, _ := legacyAttr(k)
		v.CopyTo(attrs.PutEmpty(nk))
		attrs.Remove(k)
	}
}

func forEachAttributes(m pmetric.Metric, f func(pcommon.Map)) {
	switch m.Type() {
	case pmetric.MetricTypeGauge:
		for i := 0; i < m.Gauge().DataPoints().Len(); i++ {
			f(m.Gauge().DataPoints().At(i).Attributes())
		}
	case pmetric.MetricTypeSum:
		for i := 0; i < m.Sum().DataPoints().Len(); i++ {
			f(m.Sum().DataPoints().At(i).Attributes())
		}
	case pmetric.MetricTypeHistogram:
		for i := 0; i < m.Histogram().DataPoints().Len(); i++ {
			f(m.Histogram().DataPoints().At(i).Attributes())
		}
	case pmetric.MetricTypeExponentialHistogram:
		for i := 0; i < m.ExponentialHistogram().DataPoints().Len(); i++ {
			f(m.ExponentialHistogram().DataPoints().At(i).Attributes())
		}
	case pmetric.MetricTypeSummary:
		for i := 0; i < m.Summary().DataPoints().Len(); i++ {
			f(m.Summary().DataPoints().At(i).Attributes())
		}
	}
}
