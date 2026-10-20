// Stanza operators registry dedicated to ByLonis logs pipelines

package bylonislogspipelinestanzaoperator

import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"

var StanzaOperatorsRegistry = operator.NewRegistry()

// Register will register an operator in the default registry
func Register(operatorType string, newBuilder func() operator.Builder) {
	StanzaOperatorsRegistry.Register(operatorType, newBuilder)
}

// Lookup looks up a given operator type.Its second return value will
// be false if no builder is registered for that type.
func Lookup(configType string) (func() operator.Builder, bool) {
	return StanzaOperatorsRegistry.Lookup(configType)
}
