package previewserver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bylonis/bylonis-otel-collector/pkg/collectorsimulator"
	"github.com/knadh/koanf/parsers/yaml"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
)

// Processor is one processor of the logs pipeline, as the backend generates it for the pipelines.
type Processor struct {
	// Name is the component ID, e.g. bylonislogspipeline/pipeline_nginx.
	Name string `json:"name"`
	// Config is literal: ${...} is not expanded.
	Config map[string]any `json:"config"`
}

// escapeDollars returns config with every $ written as $$, the way the backend escapes the
// processors it pushes over OpAMP, so the collector config loader turns them back into $ and
// never expands ${env:...} or ${file:...} from the request.
func escapeDollars(config map[string]any) (map[string]any, error) {
	if config == nil {
		return nil, nil
	}
	serialized, err := yaml.Parser().Marshal(config)
	if err != nil {
		return nil, err
	}
	escaped := strings.ReplaceAll(string(serialized), "$", "$$")
	return yaml.Parser().Unmarshal([]byte(escaped))
}

// configGenerator adds processors, in order, to the logs pipeline of the simulation config.
func configGenerator(processors []Processor) collectorsimulator.ConfigGenerator {
	return func(baseConf []byte) ([]byte, error) {
		conf, err := yaml.Parser().Unmarshal(baseConf)
		if err != nil {
			return nil, err
		}

		processorConfs := map[string]any{}
		names := make([]any, 0, len(processors))
		for _, p := range processors {
			escaped, err := escapeDollars(p.Config)
			if err != nil {
				return nil, fmt.Errorf("processor %s: %w", p.Name, err)
			}
			processorConfs[p.Name] = escaped
			names = append(names, p.Name)
		}
		conf["processors"] = processorConfs

		service, ok := conf["service"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("simulation config has no service")
		}
		pipelines, ok := service["pipelines"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("simulation config has no pipelines")
		}
		logsPipeline, ok := pipelines["logs"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("simulation config has no logs pipeline")
		}
		logsPipeline["processors"] = names

		return yaml.Parser().Marshal(conf)
	}
}

// splitLogs returns one plog.Logs per log record, with its resource and scope, the way the
// backend used to feed the simulator.
func splitLogs(logs plog.Logs) []plog.Logs {
	result := []plog.Logs{}
	for i := 0; i < logs.ResourceLogs().Len(); i++ {
		rl := logs.ResourceLogs().At(i)
		for j := 0; j < rl.ScopeLogs().Len(); j++ {
			sl := rl.ScopeLogs().At(j)
			for k := 0; k < sl.LogRecords().Len(); k++ {
				single := plog.NewLogs()
				singleRL := single.ResourceLogs().AppendEmpty()
				rl.Resource().CopyTo(singleRL.Resource())
				singleRL.SetSchemaUrl(rl.SchemaUrl())
				singleSL := singleRL.ScopeLogs().AppendEmpty()
				sl.Scope().CopyTo(singleSL.Scope())
				singleSL.SetSchemaUrl(sl.SchemaUrl())
				sl.LogRecords().At(k).CopyTo(singleSL.LogRecords().AppendEmpty())
				result = append(result, single)
			}
		}
	}
	return result
}

// simulate runs logs through the processors in an in-memory collector (receiver → processors →
// exporter) and returns the processed logs, merged, and the warnings and errors it logged.
func simulate(
	ctx context.Context,
	factories map[component.Type]processor.Factory,
	processors []Processor,
	logs plog.Logs,
	timeout time.Duration,
) (plog.Logs, []string, error) {
	simulator, initCleanup, err := collectorsimulator.NewCollectorSimulator(ctx, factories, configGenerator(processors))
	if initCleanup != nil {
		defer initCleanup()
	}
	if err != nil {
		return plog.Logs{}, nil, err
	}

	startCleanup, err := simulator.Start(ctx)
	// The in-memory components are indexed globally once started: always clean them up.
	if startCleanup != nil {
		defer startCleanup()
	}
	if err != nil {
		_, _ = simulator.Shutdown(ctx)
		return plog.Logs{}, nil, collectorsimulator.InvalidConfigError(err)
	}

	expected := logs.LogRecordCount()
	for _, single := range splitLogs(logs) {
		if err := collectorsimulator.SendLogsToSimulator(ctx, simulator, single); err != nil {
			_, _ = simulator.Shutdown(ctx)
			return plog.Logs{}, nil, fmt.Errorf("could not consume logs for simulation: %w", err)
		}
	}

	// The pipeline processors batch internally, so wait until every record came through or the
	// timeout passes.
	exporter := simulator.GetExporter()
	if exporter == nil {
		_, _ = simulator.Shutdown(ctx)
		return plog.Logs{}, nil, fmt.Errorf("could not find in memory exporter for simulator")
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) && exportedRecords(exporter.GetLogs()) < expected && ctx.Err() == nil {
		time.Sleep(20 * time.Millisecond)
	}
	exported := exporter.GetLogs()

	simulationErrs, err := simulator.Shutdown(ctx)
	if err != nil {
		return plog.Logs{}, nil, fmt.Errorf("could not shutdown logs processing simulator: %w", err)
	}

	collectorLogs := []string{}
	for _, l := range simulationErrs {
		// featuregate.go logs about the feature gates of the process, not about the pipelines.
		if l == "" || strings.Contains(l, "featuregate.go") {
			continue
		}
		collectorLogs = append(collectorLogs, l)
	}

	merged := plog.NewLogs()
	for _, l := range exported {
		l.ResourceLogs().MoveAndAppendTo(merged.ResourceLogs())
	}
	return merged, collectorLogs, nil
}

func exportedRecords(logs []plog.Logs) int {
	n := 0
	for _, l := range logs {
		n += l.LogRecordCount()
	}
	return n
}
