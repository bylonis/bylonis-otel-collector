package dbnames

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckEnv(t *testing.T) {
	for _, tt := range []struct {
		name    string
		env     map[string]string
		wantErr bool
	}{
		{name: "unset"},
		{name: "empty", env: map[string]string{RemovedPrefixEnv: ""}},
		{name: "bylonis", env: map[string]string{RemovedPrefixEnv: "bylonis"}},
		{name: "upstream prefix", env: map[string]string{RemovedPrefixEnv: "upstream"}, wantErr: true},
		{name: "other prefix", env: map[string]string{RemovedPrefixEnv: "acme"}, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckEnv(func(k string) (string, bool) { v, ok := tt.env[k]; return v, ok })
			if !tt.wantErr {
				assert.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, RemovedPrefixEnv)
			assert.ErrorContains(t, err, "bylonis_traces")
		})
	}
}

func TestNames(t *testing.T) {
	assert.Equal(t, []string{"bylonis_traces", "bylonis_metrics", "bylonis_logs", "bylonis_meter", "bylonis_metadata", "bylonis_analytics"}, All)
}
