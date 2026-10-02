// Package dbprefix makes the prefix of the ClickHouse database names
// configurable. The code names the databases bylonis_traces, bylonis_metrics,
// bylonis_logs, bylonis_meter, bylonis_metadata and bylonis_analytics, both as
// Go constants and inside hundreds of SQL strings (schema migrations,
// exporters). Upstream SigNoz calls them signoz_<db>.
//
// The prefix comes from the BYLONIS_DB_PREFIX env var and defaults to
// "bylonis". Connections opened with Open rewrite both the bylonis_<db> and the
// legacy signoz_<db> names in every statement and string argument to the
// configured prefix. That keeps SQL saved with the upstream names working, and
// BYLONIS_DB_PREFIX=signoz serves installs whose databases were never renamed.
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
	// Default is the prefix of the database names in the code.
	Default = "bylonis"
	// Legacy is the upstream SigNoz prefix, still accepted in SQL.
	Legacy = "signoz"
)

// suffixes are the database names after "<prefix>_".
var suffixes = []string{"traces", "metrics", "logs", "meter", "metadata", "analytics"}

var (
	validPrefix = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)
	knownNames  = regexp.MustCompile(`\b(?:` + Default + `|` + Legacy + `)_(` + strings.Join(suffixes, "|") + `)\b`)

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

// IsDefault reports whether the prefix is the one the code uses. Even then,
// legacy signoz_<db> names are rewritten.
func IsDefault() bool { return prefix == Default }

func name(suffix string) string { return prefix + "_" + suffix }

// Database names for the configured prefix.
func Traces() string    { return name("traces") }
func Metrics() string   { return name("metrics") }
func Logs() string      { return name("logs") }
func Meter() string     { return name("meter") }
func Metadata() string  { return name("metadata") }
func Analytics() string { return name("analytics") }

// Rewrite replaces the bylonis_<db> and legacy signoz_<db> names in s with the
// configured prefix. Table names such as signoz_index_v3 or signoz_logs_v2 are
// left alone: only whole words matching a database name change.
func Rewrite(s string) string {
	return rewriteWith(prefix, s)
}

func rewriteWith(p, s string) string {
	// Fast path: nothing to rewrite unless s holds a name with another prefix.
	if !strings.Contains(s, Legacy+"_") && (p == Default || !strings.Contains(s, Default+"_")) {
		return s
	}
	return knownNames.ReplaceAllString(s, p+"_$1")
}
