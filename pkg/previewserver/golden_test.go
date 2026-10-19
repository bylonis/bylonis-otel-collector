package previewserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
)

// The golden cases in testdata/pipelines come from the logs pipelines tests of the ByLonis backend
// (bylonis/bylonis, pkg/query-service/app/logparsingpipeline): each file is one test, each case one
// preview it asked for, with the processors the backend generated (its testdata/preview/<Test>.yaml)
// and the logs it sent. The responses were recorded with the in-process simulator the backend used
// before the preview moved here, so they pin the behaviour the backend tests checked.
//
// Records are compared by their input order (the __bylonis_input_idx__ attribute the backend adds),
// without the observed timestamp, which is the processing time.

const inputOrderAttribute = "__bylonis_input_idx__"

type goldenFile struct {
	Source string `json:"source"`
	Cases  []struct {
		Request  json.RawMessage `json:"request"`
		Response Response        `json:"response"`
	} `json:"cases"`
}

// canonicalLogs flattens OTLP/JSON logs into one map per record, sorted by input order.
func canonicalLogs(t *testing.T, otlpJSON []byte) []map[string]any {
	t.Helper()
	logs, err := (&plog.JSONUnmarshaler{}).UnmarshalLogs(otlpJSON)
	require.NoError(t, err)

	records := []map[string]any{}
	for i := 0; i < logs.ResourceLogs().Len(); i++ {
		rl := logs.ResourceLogs().At(i)
		for j := 0; j < rl.ScopeLogs().Len(); j++ {
			sl := rl.ScopeLogs().At(j)
			for k := 0; k < sl.LogRecords().Len(); k++ {
				r := sl.LogRecords().At(k)
				records = append(records, map[string]any{
					"resource":        rl.Resource().Attributes().AsRaw(),
					"scope":           sl.Scope().Name(),
					"time":            r.Timestamp().String(),
					"severity_text":   r.SeverityText(),
					"severity_number": int32(r.SeverityNumber()),
					"body":            r.Body().AsRaw(),
					"attributes":      r.Attributes().AsRaw(),
					"trace_id":        r.TraceID().String(),
					"span_id":         r.SpanID().String(),
					"flags":           uint32(r.Flags()),
				})
			}
		}
	}
	sort.SliceStable(records, func(a, b int) bool {
		ia, _ := records[a]["attributes"].(map[string]any)[inputOrderAttribute].(int64)
		ib, _ := records[b]["attributes"].(map[string]any)[inputOrderAttribute].(int64)
		return ia < ib
	})
	return records
}

func TestGoldenPipelines(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "pipelines", "*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, files)

	limits := DefaultLimits()
	limits.MaxConcurrent = 1000
	srv := newTestServer(t, limits)
	handler := srv.Handler()

	for _, file := range files {
		content, err := os.ReadFile(file)
		require.NoError(t, err)
		var golden goldenFile
		require.NoError(t, json.Unmarshal(content, &golden))
		require.NotEmpty(t, golden.Cases, file)

		name := strings.TrimSuffix(filepath.Base(file), ".json")
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for i, c := range golden.Cases {
				req := httptest.NewRequest(http.MethodPost, Path, bytes.NewReader(c.Request))
				req.Header.Set("Authorization", "Bearer "+testToken)
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				require.Equal(t, http.StatusOK, rec.Code, "case %d: %s", i, rec.Body.String())

				var got Response
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
				require.Equal(t, canonicalLogs(t, c.Response.Logs), canonicalLogs(t, got.Logs), "case %d", i)
				require.Equal(t, len(c.Response.CollectorLogs), len(got.CollectorLogs), "case %d: %v", i, got.CollectorLogs)
			}
		})
	}
}
