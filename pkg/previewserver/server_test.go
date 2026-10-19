package previewserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bylonis/bylonis-otel-collector/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

const testToken = "test-token-0123456789abcdef"

func newTestServer(t *testing.T, limits Limits) *Server {
	t.Helper()
	factories, err := components.Components()
	require.NoError(t, err)
	srv, err := New(testToken, limits, factories.Processors, zap.NewNop())
	require.NoError(t, err)
	return srv
}

func operatorsProcessor(name string, operators ...map[string]any) Processor {
	ops := make([]any, 0, len(operators))
	for _, op := range operators {
		ops = append(ops, op)
	}
	return Processor{Name: name, Config: map[string]any{"operators": ops}}
}

func addProcessor(name, field, value string) Processor {
	return operatorsProcessor(name, map[string]any{"type": "add", "id": "add", "field": field, "value": value})
}

// testLogs returns OTLP/JSON with n records, body "log <i>" and the input order attribute.
func testLogs(t *testing.T, n int, body func(i int) string) json.RawMessage {
	t.Helper()
	logs := plog.NewLogs()
	for i := 0; i < n; i++ {
		r := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
		if body != nil {
			r.Body().SetStr(body(i))
		} else {
			r.Body().SetStr(fmt.Sprintf("log %d", i))
		}
		r.Attributes().PutInt(inputOrderAttribute, int64(i))
	}
	content, err := (&plog.JSONMarshaler{}).MarshalLogs(logs)
	require.NoError(t, err)
	return content
}

func doRequest(t *testing.T, srv *Server, token string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, Path, bytes.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func previewBody(t *testing.T, processors []Processor, logs json.RawMessage, timeoutMs int64) []byte {
	t.Helper()
	body, err := json.Marshal(Request{Processors: processors, Logs: logs, TimeoutMs: timeoutMs})
	require.NoError(t, err)
	return body
}

func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder) (plog.Logs, []string) {
	t.Helper()
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var resp Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	logs, err := (&plog.JSONUnmarshaler{}).UnmarshalLogs(resp.Logs)
	require.NoError(t, err)
	return logs, resp.CollectorLogs
}

func errorMessage(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var resp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp), rec.Body.String())
	return resp.Error
}

func allRecords(logs plog.Logs) []plog.LogRecord {
	records := []plog.LogRecord{}
	for i := 0; i < logs.ResourceLogs().Len(); i++ {
		sls := logs.ResourceLogs().At(i).ScopeLogs()
		for j := 0; j < sls.Len(); j++ {
			lrs := sls.At(j).LogRecords()
			for k := 0; k < lrs.Len(); k++ {
				records = append(records, lrs.At(k))
			}
		}
	}
	return records
}

func TestNew(t *testing.T) {
	factories, err := components.Components()
	require.NoError(t, err)

	_, err = New("short", DefaultLimits(), factories.Processors, zap.NewNop())
	assert.ErrorContains(t, err, "at least")

	_, err = New(testToken, DefaultLimits(), map[component.Type]processor.Factory{}, zap.NewNop())
	assert.ErrorContains(t, err, "not registered")
}

func TestAuth(t *testing.T) {
	srv := newTestServer(t, DefaultLimits())
	body := previewBody(t, []Processor{addProcessor("bylonislogspipeline/a", "attributes.x", "y")}, testLogs(t, 1, nil), 500)

	for name, header := range map[string]string{
		"no header":    "",
		"wrong token":  "Bearer " + testToken + "x",
		"prefix only":  "Bearer " + testToken[:len(testToken)-1],
		"empty bearer": "Bearer ",
		"basic":        "Basic " + testToken,
		"raw token":    testToken,
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, Path, bytes.NewReader(body))
			if header != "" {
				req.Header.Set("Authorization", header)
			}
			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, req)
			assert.Equal(t, http.StatusUnauthorized, rec.Code)
			assert.Equal(t, "Bearer", rec.Header().Get("WWW-Authenticate"))
		})
	}

	logs, _ := decodeResponse(t, doRequest(t, srv, testToken, body))
	records := allRecords(logs)
	require.Len(t, records, 1)
	v, ok := records[0].Attributes().Get("x")
	require.True(t, ok)
	assert.Equal(t, "y", v.Str())
}

func TestMethodAndPath(t *testing.T) {
	srv := newTestServer(t, DefaultLimits())

	req := httptest.NewRequest(http.MethodGet, Path, nil)
	req.Header.Set("Authorization", "Bearer "+testToken)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)

	req = httptest.NewRequest(http.MethodPost, "/v1/other", nil)
	req.Header.Set("Authorization", "Bearer "+testToken)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestLimits(t *testing.T) {
	limits := DefaultLimits()
	limits.MaxBodyBytes = 4 << 10
	limits.MaxLogs = 2
	limits.MaxProcessors = 1
	srv := newTestServer(t, limits)
	one := []Processor{addProcessor("bylonislogspipeline/a", "attributes.x", "y")}

	t.Run("body", func(t *testing.T) {
		big := strings.Repeat("x", int(limits.MaxBodyBytes))
		rec := doRequest(t, srv, testToken, previewBody(t, one, testLogs(t, 1, func(int) string { return big }), 500))
		assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	})
	t.Run("logs", func(t *testing.T) {
		rec := doRequest(t, srv, testToken, previewBody(t, one, testLogs(t, 3, nil), 500))
		assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
		assert.Contains(t, errorMessage(t, rec), "3 logs")
	})
	t.Run("processors", func(t *testing.T) {
		two := append(one, addProcessor("bylonislogspipeline/b", "attributes.z", "y"))
		rec := doRequest(t, srv, testToken, previewBody(t, two, testLogs(t, 1, nil), 500))
		assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	})
	t.Run("at the limits", func(t *testing.T) {
		logs, _ := decodeResponse(t, doRequest(t, srv, testToken, previewBody(t, one, testLogs(t, 2, nil), 500)))
		assert.Len(t, allRecords(logs), 2)
	})
}

func TestBusy(t *testing.T) {
	limits := DefaultLimits()
	limits.MaxConcurrent = 2
	srv := newTestServer(t, limits)
	body := previewBody(t, []Processor{addProcessor("bylonislogspipeline/a", "attributes.x", "y")}, testLogs(t, 1, nil), 500)

	// Two previews in flight.
	srv.slots <- struct{}{}
	srv.slots <- struct{}{}
	rec := doRequest(t, srv, testToken, body)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	<-srv.slots
	decodeResponse(t, doRequest(t, srv, testToken, body))
	<-srv.slots
}

func TestRejectedProcessors(t *testing.T) {
	srv := newTestServer(t, DefaultLimits())
	logs := testLogs(t, 1, nil)
	validOps := addProcessor("x", "attributes.x", "y").Config

	for name, processors := range map[string][]Processor{
		"batch processor":         {{Name: "batch", Config: map[string]any{}}},
		"logstransform processor": {{Name: "logstransform/pipeline_a", Config: validOps}},
		"transform processor":     {{Name: "transform/a", Config: map[string]any{}}},
		"exporter type":           {{Name: "file/a", Config: map[string]any{"path": "/tmp/out"}}},
		"legacy type":             {{Name: "signozlogspipeline/pipeline_a", Config: validOps}},
		"empty name":              {{Name: "", Config: validOps}},
		"invalid name":            {{Name: "bylonislogspipeline/", Config: validOps}},
		"repeated":                {{Name: "bylonislogspipeline/a", Config: validOps}, {Name: "bylonislogspipeline/a", Config: validOps}},
	} {
		t.Run(name, func(t *testing.T) {
			rec := doRequest(t, srv, testToken, previewBody(t, processors, logs, 500))
			assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		})
	}
}

func TestInvalidRequests(t *testing.T) {
	srv := newTestServer(t, DefaultLimits())
	logs := testLogs(t, 1, nil)

	for name, body := range map[string][]byte{
		"not json":     []byte("{"),
		"invalid logs": []byte(`{"processors": [], "logs": {"resourceLogs": 3}}`),
		"unknown operator type": previewBody(t, []Processor{operatorsProcessor("bylonislogspipeline/a",
			map[string]any{"type": "no_such_operator", "id": "x"})}, logs, 500),
		"no operators": previewBody(t, []Processor{{Name: "bylonislogspipeline/a", Config: map[string]any{"operators": []any{}}}}, logs, 500),
		"null config":  previewBody(t, []Processor{{Name: "bylonislogspipeline/a"}}, logs, 500),
		"invalid regex": previewBody(t, []Processor{operatorsProcessor("bylonislogspipeline/a",
			map[string]any{"type": "regex_parser", "id": "r", "regex": "(?P<x>[", "parse_from": "body"})}, logs, 500),
		"unknown output": previewBody(t, []Processor{operatorsProcessor("bylonislogspipeline/a",
			map[string]any{"type": "add", "id": "add", "field": "attributes.x", "value": "y", "output": "nowhere"})}, logs, 500),
	} {
		t.Run(name, func(t *testing.T) {
			rec := doRequest(t, srv, testToken, body)
			assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			assert.NotEmpty(t, errorMessage(t, rec))
		})
	}
}

func TestProcessorsRunInOrder(t *testing.T) {
	srv := newTestServer(t, DefaultLimits())
	processors := []Processor{
		addProcessor("bylonislogspipeline/first", "attributes.a", "1"),
		operatorsProcessor("bylonislogspipeline/second", map[string]any{"type": "copy", "id": "copy", "from": "attributes.a", "to": "attributes.b"}),
	}

	start := time.Now()
	// A long timeout does not delay the answer once every record came through.
	logs, collectorLogs := decodeResponse(t, doRequest(t, srv, testToken, previewBody(t, processors, testLogs(t, 3, nil), 60_000)))
	assert.Less(t, time.Since(start), 5*time.Second)
	assert.Empty(t, collectorLogs)

	records := allRecords(logs)
	require.Len(t, records, 3)
	for _, r := range records {
		b, ok := r.Attributes().Get("b")
		require.True(t, ok)
		assert.Equal(t, "1", b.Str())
	}
}

func TestDollarsAreNotExpanded(t *testing.T) {
	t.Setenv("PREVIEW_TEST_SECRET", "leaked")
	srv := newTestServer(t, DefaultLimits())

	values := []string{"${env:PREVIEW_TEST_SECRET}", "${PREVIEW_TEST_SECRET}", "$PREVIEW_TEST_SECRET", "${file:/etc/hostname}", "a$$b"}
	processors := []Processor{}
	for i, v := range values {
		processors = append(processors, addProcessor(fmt.Sprintf("bylonislogspipeline/p%d", i), fmt.Sprintf("attributes.v%d", i), v))
	}

	logs, _ := decodeResponse(t, doRequest(t, srv, testToken, previewBody(t, processors, testLogs(t, 1, nil), 2000)))
	records := allRecords(logs)
	require.Len(t, records, 1)
	for i, v := range values {
		got, ok := records[0].Attributes().Get(fmt.Sprintf("v%d", i))
		require.True(t, ok)
		assert.Equal(t, v, got.Str())
	}
}

func TestCollectorLogsAreReturned(t *testing.T) {
	srv := newTestServer(t, DefaultLimits())
	// on_error: send logs the failure and passes the entry on.
	processors := []Processor{operatorsProcessor("bylonislogspipeline/a",
		map[string]any{"type": "json_parser", "id": "json", "parse_from": "body", "on_error": "send"})}

	logs, collectorLogs := decodeResponse(t, doRequest(t, srv, testToken, previewBody(t, processors, testLogs(t, 1, nil), 1000)))
	assert.Len(t, allRecords(logs), 1)
	assert.NotEmpty(t, collectorLogs)
}

func TestNoLogs(t *testing.T) {
	srv := newTestServer(t, DefaultLimits())
	processors := []Processor{addProcessor("bylonislogspipeline/a", "attributes.x", "y")}

	logs, _ := decodeResponse(t, doRequest(t, srv, testToken, previewBody(t, processors, nil, 500)))
	assert.Equal(t, 0, logs.LogRecordCount())

	// Validation only: no logs and an invalid config.
	rec := doRequest(t, srv, testToken, previewBody(t, []Processor{{Name: "bylonislogspipeline/a"}}, nil, 500))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestStartAndShutdown(t *testing.T) {
	srv := newTestServer(t, DefaultLimits())
	require.NoError(t, srv.Start("127.0.0.1:0"))
	require.NoError(t, srv.Shutdown(t.Context()))
}
