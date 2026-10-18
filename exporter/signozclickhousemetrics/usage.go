package signozclickhousemetrics

import (
	"fmt"
	"github.com/bylonis/bylonis-otel-collector/usage"
	"github.com/google/uuid"
	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"strings"
)

const (
	BylonisSentMetricPointsKey      = "bylonis_sent_metric_points"
	BylonisSentMetricPointsBytesKey = "bylonis_sent_metric_points_bytes"
	BylonisMetricPointsCount        = "bylonis_metric_points_count"
	BylonisMetricPointsBytes        = "bylonis_metric_points_bytes"
)

var (
	// Measures for usage
	ExporterBylonisSentMetricPoints = stats.Int64(
		BylonisSentMetricPointsKey,
		"Number of metric points successfully sent to destination.",
		stats.UnitDimensionless)
	ExporterBylonisSentMetricPointsBytes = stats.Int64(
		BylonisSentMetricPointsBytesKey,
		"Total size of metric points successfully sent to destination.",
		stats.UnitDimensionless)

	// Views for usage
	MetricPointsCountView = &view.View{
		Name:        BylonisMetricPointsCount,
		Measure:     ExporterBylonisSentMetricPoints,
		Description: "The number of metric points exported to ByLonis",
		Aggregation: view.Sum(),
		TagKeys:     []tag.Key{usage.TagTenantKey, usage.TagExporterIdKey},
	}
	MetricPointsBytesView = &view.View{
		Name:        BylonisMetricPointsBytes,
		Measure:     ExporterBylonisSentMetricPointsBytes,
		Description: "The size of metric points exported to ByLonis",
		Aggregation: view.Sum(),
		TagKeys:     []tag.Key{usage.TagTenantKey, usage.TagExporterIdKey},
	}
)

func UsageExporter(metrics []*metricdata.Metric, id uuid.UUID) (map[string]usage.Usage, error) {
	data := map[string]usage.Usage{}
	for _, metric := range metrics {
		if !strings.Contains(metric.Descriptor.Name, BylonisMetricPointsCount) && !strings.Contains(metric.Descriptor.Name, BylonisMetricPointsBytes) {
			continue
		}
		exporterIndex := usage.GetIndexOfLabel(metric.Descriptor.LabelKeys, usage.ExporterIDKey)
		tenantIndex := usage.GetIndexOfLabel(metric.Descriptor.LabelKeys, usage.TenantKey)
		if exporterIndex == -1 || tenantIndex == -1 {
			return nil, fmt.Errorf("usage: failed to get index of labels")
		}
		if strings.Contains(metric.Descriptor.Name, BylonisMetricPointsCount) {
			for _, v := range metric.TimeSeries {
				if v.LabelValues[exporterIndex].Value != id.String() {
					continue
				}
				tenant := v.LabelValues[tenantIndex].Value
				if d, ok := data[tenant]; ok {
					d.Count = v.Points[0].Value.(int64)
					data[tenant] = d
				} else {
					data[tenant] = usage.Usage{
						Count: v.Points[0].Value.(int64),
					}
				}
			}
		} else if strings.Contains(metric.Descriptor.Name, BylonisMetricPointsBytes) {
			for _, v := range metric.TimeSeries {
				if v.LabelValues[exporterIndex].Value != id.String() {
					continue
				}
				tenant := v.LabelValues[tenantIndex].Value
				if d, ok := data[tenant]; ok {
					d.Size = v.Points[0].Value.(int64)
					data[tenant] = d
				} else {
					data[tenant] = usage.Usage{
						Size: v.Points[0].Value.(int64),
					}
				}
			}
		}
	}
	return data, nil
}
