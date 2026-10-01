package components

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
)

const (
	legacyTypePrefix = "signoz"
	aliasTypePrefix  = "bylonis"
)

// Every component whose type starts with "signoz" (signozlogspipeline,
// signozspanmetrics, signoz_health_check, ...) is also registered under the
// same name with "bylonis" instead, e.g. bylonislogspipeline. Both names build
// the same component, so existing configs, including the ones the ByLonis
// backend pushes over OpAMP, keep working while new configs use bylonis*.

// The original factories check that the component ID has their own type, so
// each alias hands them the ID with the original type and the same name.
func originalID(id component.ID, f component.Factory) component.ID {
	return component.NewIDWithName(f.Type(), id.Name())
}

type aliasedReceiver struct {
	receiver.Factory
	typ component.Type
}

func (f aliasedReceiver) Type() component.Type { return f.typ }

func (f aliasedReceiver) CreateTraces(ctx context.Context, set receiver.Settings, cfg component.Config, next consumer.Traces) (receiver.Traces, error) {
	set.ID = originalID(set.ID, f.Factory)
	return f.Factory.CreateTraces(ctx, set, cfg, next)
}

func (f aliasedReceiver) CreateMetrics(ctx context.Context, set receiver.Settings, cfg component.Config, next consumer.Metrics) (receiver.Metrics, error) {
	set.ID = originalID(set.ID, f.Factory)
	return f.Factory.CreateMetrics(ctx, set, cfg, next)
}

func (f aliasedReceiver) CreateLogs(ctx context.Context, set receiver.Settings, cfg component.Config, next consumer.Logs) (receiver.Logs, error) {
	set.ID = originalID(set.ID, f.Factory)
	return f.Factory.CreateLogs(ctx, set, cfg, next)
}

type aliasedProcessor struct {
	processor.Factory
	typ component.Type
}

func (f aliasedProcessor) Type() component.Type { return f.typ }

func (f aliasedProcessor) CreateTraces(ctx context.Context, set processor.Settings, cfg component.Config, next consumer.Traces) (processor.Traces, error) {
	set.ID = originalID(set.ID, f.Factory)
	return f.Factory.CreateTraces(ctx, set, cfg, next)
}

func (f aliasedProcessor) CreateMetrics(ctx context.Context, set processor.Settings, cfg component.Config, next consumer.Metrics) (processor.Metrics, error) {
	set.ID = originalID(set.ID, f.Factory)
	return f.Factory.CreateMetrics(ctx, set, cfg, next)
}

func (f aliasedProcessor) CreateLogs(ctx context.Context, set processor.Settings, cfg component.Config, next consumer.Logs) (processor.Logs, error) {
	set.ID = originalID(set.ID, f.Factory)
	return f.Factory.CreateLogs(ctx, set, cfg, next)
}

type aliasedExporter struct {
	exporter.Factory
	typ component.Type
}

func (f aliasedExporter) Type() component.Type { return f.typ }

func (f aliasedExporter) CreateTraces(ctx context.Context, set exporter.Settings, cfg component.Config) (exporter.Traces, error) {
	set.ID = originalID(set.ID, f.Factory)
	return f.Factory.CreateTraces(ctx, set, cfg)
}

func (f aliasedExporter) CreateMetrics(ctx context.Context, set exporter.Settings, cfg component.Config) (exporter.Metrics, error) {
	set.ID = originalID(set.ID, f.Factory)
	return f.Factory.CreateMetrics(ctx, set, cfg)
}

func (f aliasedExporter) CreateLogs(ctx context.Context, set exporter.Settings, cfg component.Config) (exporter.Logs, error) {
	set.ID = originalID(set.ID, f.Factory)
	return f.Factory.CreateLogs(ctx, set, cfg)
}

type aliasedExtension struct {
	extension.Factory
	typ component.Type
}

func (f aliasedExtension) Type() component.Type { return f.typ }

func (f aliasedExtension) Create(ctx context.Context, set extension.Settings, cfg component.Config) (extension.Extension, error) {
	set.ID = originalID(set.ID, f.Factory)
	return f.Factory.Create(ctx, set, cfg)
}

type aliasedConnector struct {
	connector.Factory
	typ component.Type
}

func (f aliasedConnector) Type() component.Type { return f.typ }

func (f aliasedConnector) CreateTracesToTraces(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Traces) (connector.Traces, error) {
	set.ID = originalID(set.ID, f.Factory)
	return f.Factory.CreateTracesToTraces(ctx, set, cfg, next)
}

func (f aliasedConnector) CreateTracesToMetrics(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Metrics) (connector.Traces, error) {
	set.ID = originalID(set.ID, f.Factory)
	return f.Factory.CreateTracesToMetrics(ctx, set, cfg, next)
}

func (f aliasedConnector) CreateTracesToLogs(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Logs) (connector.Traces, error) {
	set.ID = originalID(set.ID, f.Factory)
	return f.Factory.CreateTracesToLogs(ctx, set, cfg, next)
}

func (f aliasedConnector) CreateMetricsToTraces(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Traces) (connector.Metrics, error) {
	set.ID = originalID(set.ID, f.Factory)
	return f.Factory.CreateMetricsToTraces(ctx, set, cfg, next)
}

func (f aliasedConnector) CreateMetricsToMetrics(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Metrics) (connector.Metrics, error) {
	set.ID = originalID(set.ID, f.Factory)
	return f.Factory.CreateMetricsToMetrics(ctx, set, cfg, next)
}

func (f aliasedConnector) CreateMetricsToLogs(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Logs) (connector.Metrics, error) {
	set.ID = originalID(set.ID, f.Factory)
	return f.Factory.CreateMetricsToLogs(ctx, set, cfg, next)
}

func (f aliasedConnector) CreateLogsToTraces(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Traces) (connector.Logs, error) {
	set.ID = originalID(set.ID, f.Factory)
	return f.Factory.CreateLogsToTraces(ctx, set, cfg, next)
}

func (f aliasedConnector) CreateLogsToMetrics(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Metrics) (connector.Logs, error) {
	set.ID = originalID(set.ID, f.Factory)
	return f.Factory.CreateLogsToMetrics(ctx, set, cfg, next)
}

func (f aliasedConnector) CreateLogsToLogs(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Logs) (connector.Logs, error) {
	set.ID = originalID(set.ID, f.Factory)
	return f.Factory.CreateLogsToLogs(ctx, set, cfg, next)
}

// addBylonisAliases adds a bylonis* entry for every signoz* factory in m.
func addBylonisAliases[F any](m map[component.Type]F, alias func(F, component.Type) F) error {
	for typ, factory := range m {
		name := typ.String()
		if !strings.HasPrefix(name, legacyTypePrefix) {
			continue
		}
		aliasType, err := component.NewType(aliasTypePrefix + strings.TrimPrefix(name, legacyTypePrefix))
		if err != nil {
			return fmt.Errorf("alias for %q: %w", name, err)
		}
		if _, exists := m[aliasType]; exists {
			return fmt.Errorf("alias %q for %q is already registered", aliasType, name)
		}
		m[aliasType] = alias(factory, aliasType)
	}
	return nil
}
