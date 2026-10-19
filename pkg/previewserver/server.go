// Package previewserver serves the logs pipelines preview of the ByLonis backend: it runs the
// pipeline processors the backend generates on sample logs, in an in-memory collector built with
// the same processor this binary runs for ingestion, and returns the processed logs.
//
// It runs from the binary, outside the collector config (which the backend replaces over
// OpAMP), and only when a token is configured. The endpoint is meant for the tenant's internal
// network only.
//
//	POST /v1/logs-pipelines/preview
//	Authorization: Bearer <token>
//
//	{"processors": [{"name": "bylonislogspipeline/pipeline_x", "config": {...}}],  // in execution order
//	 "logs": {...OTLP/JSON...},
//	 "timeout_ms": 1100}
//
//	200 {"logs": {...OTLP/JSON...}, "collector_logs": ["..."]}
//	400 invalid request or processor config, 401, 405, 413, 429 busy; errors as {"error": "..."}
//
// The received logs are never logged.
package previewserver

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/bylonis/bylonis-otel-collector/pkg/collectorsimulator"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

// Path is the preview endpoint. The version changes with any incompatible change of the contract.
const Path = "/v1/logs-pipelines/preview"

// ProcessorTypes are the processor types the endpoint accepts: the logs pipeline processor
// under the type the backend uses for pipelines. Anything else (exporters, receivers, other
// processors) can not be configured through the endpoint.
var ProcessorTypes = []component.Type{component.MustNewType("bylonislogspipeline")}

// MinTokenLength is the minimum length of the token.
const MinTokenLength = 16

// Limits bound what a single request can ask for.
type Limits struct {
	// MaxBodyBytes is the maximum size of the request body.
	MaxBodyBytes int64
	// MaxLogs is the maximum number of log records per request.
	MaxLogs int
	// MaxProcessors is the maximum number of processors per request.
	MaxProcessors int
	// DefaultTimeout is used when the request has no timeout.
	DefaultTimeout time.Duration
	// MaxTimeout caps the timeout of a request.
	MaxTimeout time.Duration
	// MaxConcurrent is the number of previews that run at once; more get 429.
	MaxConcurrent int
}

func DefaultLimits() Limits {
	return Limits{
		MaxBodyBytes:   1 << 20,
		MaxLogs:        100,
		MaxProcessors:  100,
		DefaultTimeout: time.Second,
		MaxTimeout:     10 * time.Second,
		MaxConcurrent:  2,
	}
}

type Request struct {
	Processors []Processor `json:"processors"`
	// Logs is OTLP/JSON (plog.JSONMarshaler).
	Logs      json.RawMessage `json:"logs"`
	TimeoutMs int64           `json:"timeout_ms"`
}

type Response struct {
	// Logs is OTLP/JSON (plog.JSONMarshaler).
	Logs          json.RawMessage `json:"logs"`
	CollectorLogs []string        `json:"collector_logs"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type Server struct {
	tokenHash [sha256.Size]byte
	limits    Limits
	factories map[component.Type]processor.Factory
	slots     chan struct{}
	logger    *zap.Logger
	srv       *http.Server
}

// New returns a preview server. allFactories are the processor factories of the collector; only
// ProcessorTypes are used.
func New(token string, limits Limits, allFactories map[component.Type]processor.Factory, logger *zap.Logger) (*Server, error) {
	if len(token) < MinTokenLength {
		return nil, fmt.Errorf("the preview token must have at least %d characters", MinTokenLength)
	}
	factories := map[component.Type]processor.Factory{}
	for _, typ := range ProcessorTypes {
		f, ok := allFactories[typ]
		if !ok {
			return nil, fmt.Errorf("processor %q is not registered", typ)
		}
		factories[typ] = f
	}
	if limits.MaxConcurrent < 1 {
		limits.MaxConcurrent = 1
	}
	return &Server{
		tokenHash: sha256.Sum256([]byte(token)),
		limits:    limits,
		factories: factories,
		slots:     make(chan struct{}, limits.MaxConcurrent),
		logger:    logger,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(Path, s.handlePreview)
	return mux
}

// Start listens on addr and serves in the background until Shutdown.
func (s *Server) Start(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("could not listen on %s: %w", addr, err)
	}
	s.srv = &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      s.limits.MaxTimeout + 30*time.Second,
		IdleTimeout:       2 * time.Minute,
		ErrorLog:          zap.NewStdLog(s.logger),
	}
	go func() {
		if err := s.srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("logs pipelines preview server stopped", zap.Error(err))
		}
	}()
	s.logger.Info("logs pipelines preview server listening", zap.String("address", ln.Addr().String()))
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.srv == nil {
		return nil
	}
	return s.srv.Shutdown(ctx)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, format string, args ...any) {
	writeJSON(w, status, ErrorResponse{Error: fmt.Sprintf(format, args...)})
}

func (s *Server) authorized(r *http.Request) bool {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		return false
	}
	// Comparing fixed-size hashes keeps the comparison constant-time also in the length.
	hash := sha256.Sum256([]byte(token))
	return subtle.ConstantTimeCompare(hash[:], s.tokenHash[:]) == 1
}

func (s *Server) handlePreview(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		w.Header().Set("WWW-Authenticate", "Bearer")
		writeError(w, http.StatusUnauthorized, "missing or invalid token")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "use POST")
		return
	}

	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	default:
		writeError(w, http.StatusTooManyRequests, "too many previews running, try again")
		return
	}

	var req Request
	body := http.MaxBytesReader(w, r.Body, s.limits.MaxBodyBytes)
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "the request body is larger than %d bytes", s.limits.MaxBodyBytes)
			return
		}
		writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	if len(req.Processors) > s.limits.MaxProcessors {
		writeError(w, http.StatusRequestEntityTooLarge, "%d processors, at most %d", len(req.Processors), s.limits.MaxProcessors)
		return
	}
	if err := s.checkProcessors(req.Processors); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}

	logs := plog.NewLogs()
	if len(req.Logs) > 0 && string(req.Logs) != "null" {
		var err error
		logs, err = (&plog.JSONUnmarshaler{}).UnmarshalLogs(req.Logs)
		if err != nil {
			// The decoder error does not echo the logs.
			writeError(w, http.StatusBadRequest, "invalid logs, expected OTLP/JSON: %v", err)
			return
		}
	}
	if n := logs.LogRecordCount(); n > s.limits.MaxLogs {
		writeError(w, http.StatusRequestEntityTooLarge, "%d logs, at most %d", n, s.limits.MaxLogs)
		return
	}

	timeout := s.limits.DefaultTimeout
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}
	timeout = min(timeout, s.limits.MaxTimeout)

	output, collectorLogs, err := simulate(r.Context(), s.factories, req.Processors, logs, timeout)
	if err != nil {
		if errors.Is(err, collectorsimulator.ErrInvalidConfig) {
			writeError(w, http.StatusBadRequest, "%v", err)
			return
		}
		s.logger.Error("logs pipelines preview failed", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "could not simulate the pipelines: %v", err)
		return
	}

	serialized, err := (&plog.JSONMarshaler{}).MarshalLogs(output)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not serialize the processed logs: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, Response{Logs: serialized, CollectorLogs: collectorLogs})
}

func (s *Server) checkProcessors(processors []Processor) error {
	seen := map[string]bool{}
	for i, p := range processors {
		var id component.ID
		if err := id.UnmarshalText([]byte(p.Name)); err != nil {
			return fmt.Errorf("processor %d: invalid name %q: %w", i, p.Name, err)
		}
		if _, ok := s.factories[id.Type()]; !ok {
			return fmt.Errorf("processor %q: type %q is not allowed, only %v", p.Name, id.Type(), ProcessorTypes)
		}
		if seen[id.String()] {
			return fmt.Errorf("processor %q is repeated", p.Name)
		}
		seen[id.String()] = true
	}
	return nil
}
