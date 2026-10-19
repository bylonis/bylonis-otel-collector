package clickhouselogsexporter

import (
	"fmt"
	"strings"

	"github.com/bylonis/bylonis-otel-collector/pkg/metering"
	"github.com/bylonis/bylonis-otel-collector/usage"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

const (
	BylonisSentLogRecordsKey      = "bylonis_sent_log_records"
	BylonisSentLogRecordsBytesKey = "bylonis_sent_log_records_bytes"
	BylonisLogsCount              = "bylonis_logs_count"
	BylonisLogsBytes              = "bylonis_logs_bytes"
)

var (
	// Measures for usage
	ExporterBylonisSentLogRecords = stats.Int64(
		BylonisSentLogRecordsKey,
		"Number of log records successfully sent to destination.",
		stats.UnitDimensionless)
	ExporterBylonisSentLogRecordsBytes = stats.Int64(
		BylonisSentLogRecordsBytesKey,
		"Total size of log records successfully sent to destination.",
		stats.UnitDimensionless)

	// Views for usage
	LogsCountView = &view.View{
		Name:        BylonisLogsCount,
		Measure:     ExporterBylonisSentLogRecords,
		Description: "The number of logs exported to ByLonis",
		Aggregation: view.Sum(),
		TagKeys:     []tag.Key{usage.TagTenantKey, usage.TagExporterIdKey},
	}
	LogsSizeView = &view.View{
		Name:        BylonisLogsBytes,
		Measure:     ExporterBylonisSentLogRecordsBytes,
		Description: "The size of logs exported to ByLonis",
		Aggregation: view.Sum(),
		TagKeys:     []tag.Key{usage.TagTenantKey, usage.TagExporterIdKey},
	}
)

func UsageExporter(metrics []*metricdata.Metric, id uuid.UUID) (map[string]usage.Usage, error) {
	data := map[string]usage.Usage{}
	for _, metric := range metrics {
		if !strings.Contains(metric.Descriptor.Name, BylonisLogsCount) && !strings.Contains(metric.Descriptor.Name, BylonisLogsBytes) {
			continue
		}
		exporterIndex := usage.GetIndexOfLabel(metric.Descriptor.LabelKeys, usage.ExporterIDKey)
		tenantIndex := usage.GetIndexOfLabel(metric.Descriptor.LabelKeys, usage.TenantKey)
		if exporterIndex == -1 || tenantIndex == -1 {
			return nil, fmt.Errorf("usage: failed to get index of labels")
		}

		if strings.Contains(metric.Descriptor.Name, BylonisLogsCount) {
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
		} else if strings.Contains(metric.Descriptor.Name, BylonisLogsBytes) {
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

func getResourceAttributesByte(resource pcommon.Resource) ([]byte, error) {
	filteredResources := map[string]any{}
	resource.Attributes().Range(func(k string, v pcommon.Value) bool {
		if !metering.ExcludeBylonisWorkspaceResourceAttrs.MatchString(k) {
			filteredResources[k] = v.AsRaw()
		}
		return true
	})
	resBytes, err := json.Marshal(filteredResources)
	if err != nil {
		return nil, err
	}

	return resBytes, nil
}
