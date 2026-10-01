// Package dbprefix makes the prefix of the ClickHouse database names
// configurable. Upstream hardcodes signoz_traces, signoz_metrics, signoz_logs,
// signoz_meter, signoz_metadata and signoz_analytics, both as Go constants and
// inside hundreds of SQL strings (schema migrations, exporters).
//
// The prefix comes from the BYLONIS_DB_PREFIX env var and defaults to
// "signoz", which keeps the upstream names and behaviour unchanged. With any
// other prefix, connections opened with Open rewrite the legacy names in every
// statement and string argument, so the SQL literals don't need to change.
package dbprefix

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

const (
	// EnvVar is the env var that sets the prefix.
	EnvVar = "BYLONIS_DB_PREFIX"
	// Default keeps the upstream database names.
	Default = "signoz"
)

// suffixes are the database names after "<prefix>_".
var suffixes = []string{"traces", "metrics", "logs", "meter", "metadata", "analytics"}

var (
	validPrefix = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)
	legacyNames = regexp.MustCompile(`\b` + Default + `_(` + strings.Join(suffixes, "|") + `)\b`)

	prefix = mustLoad(os.LookupEnv)
)

func load(lookup func(string) (string, bool)) (string, error) {
	p, ok := lookup(EnvVar)
	if !ok || p == "" {
		return Default, nil
	}
	if !validPrefix.MatchString(p) {
		return "", fmt.Errorf("invalid %s %q: must match %s", EnvVar, p, validPrefix)
	}
	return p, nil
}

func mustLoad(lookup func(string) (string, bool)) string {
	p, err := load(lookup)
	if err != nil {
		panic(err)
	}
	return p
}

// Prefix returns the configured prefix.
func Prefix() string { return prefix }

// IsDefault reports whether the prefix is the upstream one, in which case no
// rewriting happens.
func IsDefault() bool { return prefix == Default }

func name(suffix string) string { return prefix + "_" + suffix }

// Database names for the configured prefix.
func Traces() string    { return name("traces") }
func Metrics() string   { return name("metrics") }
func Logs() string      { return name("logs") }
func Meter() string     { return name("meter") }
func Metadata() string  { return name("metadata") }
func Analytics() string { return name("analytics") }

// Rewrite replaces the legacy signoz_<db> names in s with the configured
// prefix. Table names such as signoz_index_v3 or signoz_logs_v2 are left alone:
// only whole words matching a database name change.
func Rewrite(s string) string {
	return rewriteWith(prefix, s)
}

func rewriteWith(p, s string) string {
	if p == Default || !strings.Contains(s, Default+"_") {
		return s
	}
	return legacyNames.ReplaceAllString(s, p+"_$1")
}
