// Brought in as is from opentelemetry-collector-contrib
package json

import (
	"path/filepath"
	"testing"

	bylonisstanzaentry "github.com/bylonis/bylonis-otel-collector/processor/bylonislogspipelineprocessor/stanza/entry"
	bylonisstanzahelper "github.com/bylonis/bylonis-otel-collector/processor/bylonislogspipelineprocessor/stanza/operator/helper"
	"github.com/bylonis/bylonis-otel-collector/processor/bylonislogspipelineprocessor/stanza/operator/operatortest"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/entry"
)

func TestConfig(t *testing.T) {
	operatortest.ConfigUnmarshalTests{
		DefaultConfig: NewConfig(),
		TestsFile:     filepath.Join(".", "testdata", "config.yaml"),
		Tests: []operatortest.ConfigUnmarshalTest{
			{
				Name:   "default",
				Expect: NewConfig(),
			},
			{
				Name: "on_error_drop",
				Expect: func() *Config {
					cfg := NewConfig()
					cfg.OnError = "drop"
					return cfg
				}(),
			},
			{
				Name: "parse_from_simple",
				Expect: func() *Config {
					cfg := NewConfig()
					cfg.ParseFrom = bylonisstanzaentry.Field{FieldInterface: bylonisstanzaentry.NewBodyField("from")}
					return cfg
				}(),
			},
			{
				Name: "parse_to_simple",
				Expect: func() *Config {
					cfg := NewConfig()
					cfg.ParseTo = bylonisstanzaentry.RootableField{Field: bylonisstanzaentry.Field{FieldInterface: bylonisstanzaentry.NewBodyField("log")}}
					return cfg
				}(),
			},
			{
				Name: "timestamp",
				Expect: func() *Config {
					cfg := NewConfig()
					parseField := bylonisstanzaentry.Field{FieldInterface: bylonisstanzaentry.NewBodyField("timestamp_field")}
					newTime := bylonisstanzahelper.TimeParser{
						LayoutType: "strptime",
						Layout:     "%Y-%m-%d",
						ParseFrom:  &parseField,
					}
					cfg.TimeParser = &newTime
					return cfg
				}(),
			},
			{
				Name: "severity",
				Expect: func() *Config {
					cfg := NewConfig()
					parseField := bylonisstanzaentry.Field{FieldInterface: bylonisstanzaentry.NewBodyField("severity_field")}
					severityParser := bylonisstanzahelper.NewSeverityConfig()
					severityParser.ParseFrom = &parseField
					mapping := map[string]any{
						"critical": "5xx",
						"error":    "4xx",
						"info":     "3xx",
						"debug":    "2xx",
					}
					severityParser.Mapping = mapping
					cfg.SeverityConfig = &severityParser
					return cfg
				}(),
			},
			{
				Name: "scope_name",
				Expect: func() *Config {
					cfg := NewConfig()
					loggerNameParser := bylonisstanzahelper.NewScopeNameParser()
					loggerNameParser.ParseFrom = bylonisstanzaentry.Field{FieldInterface: bylonisstanzaentry.NewBodyField("logger_name_field")}
					cfg.ScopeNameParser = &loggerNameParser
					return cfg
				}(),
			},
			{
				Name: "parse_to_attributes",
				Expect: func() *Config {
					p := NewConfig()
					p.ParseTo = bylonisstanzaentry.RootableField{Field: bylonisstanzaentry.Field{FieldInterface: entry.NewAttributeField()}}
					return p
				}(),
			},
			{
				Name: "parse_to_body",
				Expect: func() *Config {
					p := NewConfig()
					p.ParseTo = bylonisstanzaentry.RootableField{Field: bylonisstanzaentry.Field{FieldInterface: bylonisstanzaentry.NewBodyField()}}
					return p
				}(),
			},
			{
				Name: "parse_to_resource",
				Expect: func() *Config {
					p := NewConfig()
					p.ParseTo = bylonisstanzaentry.RootableField{Field: bylonisstanzaentry.Field{FieldInterface: entry.NewResourceField()}}
					return p
				}(),
			},
			{
				Name: "json_flattening",
				Expect: func() *Config {
					p := NewConfig()
					p.ParseTo = bylonisstanzaentry.RootableField{Field: bylonisstanzaentry.Field{FieldInterface: entry.NewAttributeField()}}
					p.EnableFlattening = true
					p.MaxFlatteningDepth = 4
					p.EnablePaths = true
					p.PathPrefix = "parsed"
					return p
				}(),
			},
		},
	}.Run(t)
}
