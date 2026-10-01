package main

import (
	"fmt"
	"strings"

	flag "github.com/spf13/pflag"
	"go.uber.org/zap"
)

const (
	// envPrefix names the env vars that set collector flags, e.g.
	// BYLONIS_OTEL_COLLECTOR_CLICKHOUSE_DSN for --clickhouse-dsn.
	envPrefix = "BYLONIS_OTEL_COLLECTOR_"
	// legacyEnvPrefix is still read when the BYLONIS_ variable is unset.
	// Deprecated: kept so existing deployments keep working; drop in a later release.
	legacyEnvPrefix = "SIGNOZ_OTEL_COLLECTOR_"
)

func envKey(prefix, flagName string) string {
	return prefix + strings.ToUpper(strings.ReplaceAll(flagName, "-", "_"))
}

// setFlagsFromEnv sets every flag that was not given on the command line from
// its BYLONIS_OTEL_COLLECTOR_* env var, falling back to SIGNOZ_OTEL_COLLECTOR_*
// with a deprecation warning. Empty values count as unset.
func setFlagsFromEnv(flags *flag.FlagSet, lookup func(string) (string, bool), logger *zap.Logger) error {
	var errs []error
	flags.VisitAll(func(f *flag.Flag) {
		if f.Changed {
			return
		}

		key := envKey(envPrefix, f.Name)
		val, ok := lookup(key)
		if !ok || val == "" {
			legacyKey := envKey(legacyEnvPrefix, f.Name)
			val, ok = lookup(legacyKey)
			if !ok || val == "" {
				return
			}
			logger.Warn("deprecated environment variable, rename it", zap.String("old", legacyKey), zap.String("new", key))
			key = legacyKey
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
