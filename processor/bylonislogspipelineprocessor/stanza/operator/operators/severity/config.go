// Brought in as is from opentelemetry-collector-contrib

package severity

import (
	"go.opentelemetry.io/collector/component"

	bylonislogspipelinestanzaoperator "github.com/bylonis/bylonis-otel-collector/processor/bylonislogspipelineprocessor/stanza/operator"
	bylonisstanzahelper "github.com/bylonis/bylonis-otel-collector/processor/bylonislogspipelineprocessor/stanza/operator/helper"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
)

const operatorType = "severity_parser"

func init() {
	bylonislogspipelinestanzaoperator.Register(operatorType, func() operator.Builder { return NewConfig() })
}

// NewConfig creates a new severity parser config with default values
func NewConfig() *Config {
	return NewConfigWithID(operatorType)
}

// NewConfigWithID creates a new severity parser config with default values
func NewConfigWithID(operatorID string) *Config {
	return &Config{
		TransformerConfig: bylonisstanzahelper.NewTransformerConfig(operatorID, operatorType),
		SeverityConfig:    bylonisstanzahelper.NewSeverityConfig(),
	}
}

// Config is the configuration of a severity parser operator.
type Config struct {
	bylonisstanzahelper.TransformerConfig `mapstructure:",squash"`
	bylonisstanzahelper.SeverityConfig    `mapstructure:",omitempty,squash"`
}

// Build will build a severity parser operator.
func (c Config) Build(set component.TelemetrySettings) (operator.Operator, error) {
	transformerOperator, err := c.TransformerConfig.Build(set)
	if err != nil {
		return nil, err
	}

	severityParser, err := c.SeverityConfig.Build(set)
	if err != nil {
		return nil, err
	}

	return &Parser{
		TransformerOperator: transformerOperator,
		SeverityParser:      severityParser,
	}, nil
}
