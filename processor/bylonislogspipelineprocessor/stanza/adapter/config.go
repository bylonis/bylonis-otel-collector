package bylonislogspipelinestanzaadapter

import (
	bylonislogspipelinestanzaoperator "github.com/bylonis/bylonis-otel-collector/processor/bylonislogspipelineprocessor/stanza/operator"
)

type BaseConfig struct {
	// Using our own version of Config allows using a dedicated registry of stanza ops for logs pipelines.
	Operators []bylonislogspipelinestanzaoperator.Config `mapstructure:"operators"`
}
