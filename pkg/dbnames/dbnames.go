// Package dbnames names the ClickHouse databases ByLonis uses: bylonis_traces,
// bylonis_metrics, bylonis_logs, bylonis_meter, bylonis_metadata and
// bylonis_analytics. The schema migrations and the exporters use these names,
// also inside their SQL.
//
// Until v1.0 the prefix could be changed with BYLONIS_DB_PREFIX, so that
// installs whose databases still had the upstream names kept
// working, and SQL that named those was rewritten on the fly. The ByLonis
// backend only reads bylonis_<db> since v1.0, so any other prefix would leave
// the data out of its reach: the variable is gone, and CheckEnv refuses to
// start when it is still set to something else.
package dbnames

import (
	"fmt"
	"strings"
)

// Database names.
const (
	Traces    = "bylonis_traces"
	Metrics   = "bylonis_metrics"
	Logs      = "bylonis_logs"
	Meter     = "bylonis_meter"
	Metadata  = "bylonis_metadata"
	Analytics = "bylonis_analytics"
)

// RemovedPrefixEnv is the env var that set the prefix of the database names
// before v1.0.
const RemovedPrefixEnv = "BYLONIS_DB_PREFIX"

// All lists the database names.
var All = []string{Traces, Metrics, Logs, Meter, Metadata, Analytics}

// CheckEnv fails when RemovedPrefixEnv asks for databases other than the
// bylonis ones, instead of silently writing to new, empty databases. Unset,
// empty or "bylonis" (what the code always uses now) are fine.
func CheckEnv(lookup func(string) (string, bool)) error {
	p, ok := lookup(RemovedPrefixEnv)
	if !ok || p == "" || p == "bylonis" {
		return nil
	}
	return fmt.Errorf("%s=%q is no longer supported (removed in v1.0): the databases are always %s; rename them and unset the variable",
		RemovedPrefixEnv, p, strings.Join(All, ", "))
}
