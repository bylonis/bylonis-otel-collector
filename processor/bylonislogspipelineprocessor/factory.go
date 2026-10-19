// Async processor: ConsumeLogs returns immediately after enqueuing to
// FromPdataConverter. The factory therefore bypasses processorhelper.NewLogs
// (which only supports sync ProcessLogs callbacks) and returns the processor
// directly as a processor.Logs implementation.
package bylonislogspipelineprocessor

import (
	"context"
	"errors"
	"fmt"

	bylonislogspipelinestanzaadapter "github.com/bylonis/bylonis-otel-collector/processor/bylonislogspipelineprocessor/stanza/adapter"
	bylonislogspipelinestanzaoperator "github.com/bylonis/bylonis-otel-collector/processor/bylonislogspipelineprocessor/stanza/operator"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"

	"github.com/bylonis/bylonis-otel-collector/processor/bylonislogspipelineprocessor/internal/metadata"
)

func NewFactory() processor.Factory {
	return processor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		processor.WithLogs(createLogsProcessor, metadata.LogsStability))
}

// Note: This isn't a valid configuration (no operators would lead to no work being done)
func createDefaultConfig() component.Config {
	return &Config{
		BaseConfig: bylonislogspipelinestanzaadapter.BaseConfig{
			Operators: []bylonislogspipelinestanzaoperator.Config{},
		},
	}
}

func createLogsProcessor(
	_ context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	pCfg, ok := cfg.(*Config)
	if !ok {
		return nil, errors.New("could not initialize bylonislogspipeline processor")
	}
	if len(pCfg.BaseConfig.Operators) == 0 {
		return nil, errors.New("no operators were configured for bylonislogspipeline processor")
	}

	proc, err := newLogsPipelineProcessor(pCfg, set, nextConsumer)
	if err != nil {
		return nil, fmt.Errorf("couldn't build \"bylonislogspipeline\" processor %w", err)
	}

	return proc, nil
}
