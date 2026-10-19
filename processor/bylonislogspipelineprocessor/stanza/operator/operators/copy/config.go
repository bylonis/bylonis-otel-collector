// Brought in as is from opentelemetry-collector-contrib

package copy

import (
	"fmt"

	"go.opentelemetry.io/collector/component"

	bylonisstanzaentry "github.com/bylonis/bylonis-otel-collector/processor/bylonislogspipelineprocessor/stanza/entry"
	bylonislogspipelinestanzaoperator "github.com/bylonis/bylonis-otel-collector/processor/bylonislogspipelineprocessor/stanza/operator"
	bylonisstanzahelper "github.com/bylonis/bylonis-otel-collector/processor/bylonislogspipelineprocessor/stanza/operator/helper"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/entry"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
)

const operatorType = "copy"

func init() {
	bylonislogspipelinestanzaoperator.Register(operatorType, func() operator.Builder { return NewConfig() })
}

// NewConfig creates a new copy operator config with default values
func NewConfig() *Config {
	return NewConfigWithID(operatorType)
}

// NewConfigWithID creates a new copy operator config with default values
func NewConfigWithID(operatorID string) *Config {
	return &Config{
		TransformerConfig: bylonisstanzahelper.NewTransformerConfig(operatorID, operatorType),
	}
}

// Config is the configuration of a copy operator
type Config struct {
	bylonisstanzahelper.TransformerConfig `mapstructure:",squash"`
	From                                  bylonisstanzaentry.Field `mapstructure:"from"`
	To                                    entry.Field              `mapstructure:"to"`
}

// Build will build a copy operator from the supplied configuration
func (c Config) Build(set component.TelemetrySettings) (operator.Operator, error) {
	transformerOperator, err := c.TransformerConfig.Build(set)
	if err != nil {
		return nil, err
	}

	if c.From == (bylonisstanzaentry.Field{FieldInterface: entry.NewNilField()}) {
		return nil, fmt.Errorf("copy: missing from field")
	}

	if c.To == entry.NewNilField() {
		return nil, fmt.Errorf("copy: missing to field")
	}

	return &Transformer{
		TransformerOperator: transformerOperator,
		From:                c.From,
		To:                  c.To,
	}, nil
}
