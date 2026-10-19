package main

import (
	"fmt"
	"strings"

	flag "github.com/spf13/pflag"
)

const (
	// envPrefix names the env vars that set collector flags, e.g.
	// BYLONIS_OTEL_COLLECTOR_CLICKHOUSE_DSN for --clickhouse-dsn.
	envPrefix = "BYLONIS_OTEL_COLLECTOR_"
	// removedEnvPrefix names the upstream env vars, no longer read since v1.0.
	// Setting one fails instead of being silently ignored.
	removedEnvPrefix = "SIGNOZ_OTEL_COLLECTOR_"
)

func envKey(prefix, flagName string) string {
	return prefix + strings.ToUpper(strings.ReplaceAll(flagName, "-", "_"))
}

// setFlagsFromEnv sets every flag that was not given on the command line from
// its BYLONIS_OTEL_COLLECTOR_* env var. Empty values count as unset. A flag
// that is only set with the upstream prefix is an error that names the
// variable to rename.
func setFlagsFromEnv(flags *flag.FlagSet, lookup func(string) (string, bool)) error {
	var errs []error
	flags.VisitAll(func(f *flag.Flag) {
		if f.Changed {
			return
		}

		key := envKey(envPrefix, f.Name)
		val, ok := lookup(key)
		if !ok || val == "" {
			removedKey := envKey(removedEnvPrefix, f.Name)
			if old, ok := lookup(removedKey); ok && old != "" {
				errs = append(errs, fmt.Errorf("%s is no longer read (removed in v1.0): rename it to %s", removedKey, key))
			}
			return
		}

		if err := flags.Set(f.Name, val); err != nil {
			errs = append(errs, fmt.Errorf("invalid value for %s: %w", key, err))
		}
	})

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
