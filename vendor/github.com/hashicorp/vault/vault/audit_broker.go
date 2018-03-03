package vault

import (
	"context"
	"fmt"
	"sync"
	"time"

	metrics "github.com/armon/go-metrics"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/vault/audit"
	log "github.com/mgutz/logxi/v1"
)

type backendEntry struct {
	backend audit.Backend
	view    *BarrierView
}

// AuditBroker is used to provide a single ingest interface to auditable
// events given that multiple backends may be configured.
type AuditBroker struct {
	sync.RWMutex
	backends map[string]backendEntry
	logger   log.Logger
}

// NewAuditBroker creates a new audit broker
func NewAuditBroker(log log.Logger) *AuditBroker {
	b := &AuditBroker{
		backends: make(map[string]backendEntry),
		logger:   log,
	}
	return b
}

// Register is used to add new audit backend to the broker
func (a *AuditBroker) Register(name string, b audit.Backend, v *BarrierView) {
	a.Lock()
	defer a.Unlock()
	a.backends[name] = backendEntry{
		backend: b,
		view:    v,
	}
}

// Deregister is used to remove an audit backend from the broker
func (a *AuditBroker) Deregister(name string) {
	a.Lock()
	defer a.Unlock()
	delete(a.backends, name)
}

// IsRegistered is used to check if a given audit backend is registered
func (a *AuditBroker) IsRegistered(name string) bool {
	a.RLock()
	defer a.RUnlock()
	_, ok := a.backends[name]
	return ok
}

// GetHash returns a hash using the salt of the given backend
func (a *AuditBroker) GetHash(name string, input string) (string, error) {
	a.RLock()
	defer a.RUnlock()
	be, ok := a.backends[name]
	if !ok {
		return "", fmt.Errorf("unknown audit backend %s", name)
	}

	return be.backend.GetHash(input)
}

// LogRequest is used to ensure all the audit backends have an opportunity to
// log the given request and that *at least one* succeeds.
func (a *AuditBroker) LogRequest(ctx context.Context, in *audit.LogInput, headersConfig *AuditedHeadersConfig) (ret error) {
	defer metrics.MeasureSince([]string{"audit", "log_request"}, time.Now())
	a.RLock()
	defer a.RUnlock()

	var retErr *multierror.Error

	defer func() {
		if r := recover(); r != nil {
			a.logger.Error("audit: panic during logging", "request_path", in.Request.Path, "error", r)
			retErr = multierror.Append(retErr, fmt.Errorf("panic generating audit log"))
		}

		ret = retErr.ErrorOrNil()
		failure := float32(0.0)
		if ret != nil {
			failure = 1.0
		}
		metrics.IncrCounter([]string{"audit", "log_request_failure"}, failure)
	}()

	// All logged requests must have an identifier
	//if req.ID == "" {
	//	a.logger.Error("audit: missing identifier in request object", "request_path", req.Path)
	//	retErr = multierror.Append(retErr, fmt.Errorf("missing identifier in request object: %s", req.Path))
	//	return
	//}

	headers := in.Request.Headers
	defer func() {
		in.Request.Headers = headers
	}()

	// Ensure at least one backend logs
	anyLogged := false
	for name, be := range a.backends {
		in.Request.Headers = nil
		transHeaders, thErr := headersConfig.ApplyConfig(headers, be.backend.GetHash)
		if thErr != nil {
			a.logger.Error("audit: backend failed to include headers", "backend", name, "error", thErr)
			continue
		}
		in.Request.Headers = transHeaders

		start := time.Now()
		lrErr := be.backend.LogRequest(ctx, in)
		metrics.MeasureSince([]string{"audit", name, "log_request"}, start)
		if lrErr != nil {
			a.logger.Error("audit: backend failed to log request", "backend", name, "error", lrErr)
		} else {
			anyLogged = true
		}
	}
	if !anyLogged && len(a.backends) > 0 {
		retErr = multierror.Append(retErr, fmt.Errorf("no audit backend succeeded in logging the request"))
	}

	return retErr.ErrorOrNil()
}

// LogResponse is used to ensure all the audit backends have an opportunity to
// log the given response and that *at least one* succeeds.
func (a *AuditBroker) LogResponse(ctx context.Context, in *audit.LogInput, headersConfig *AuditedHeadersConfig) (ret error) {
	defer metrics.MeasureSince([]string{"audit", "log_response"}, time.Now())
	a.RLock()
	defer a.RUnlock()

	var retErr *multierror.Error

	defer func() {
		if r := recover(); r != nil {
			a.logger.Error("audit: panic during logging", "request_path", in.Request.Path, "error", r)
			retErr = multierror.Append(retErr, fmt.Errorf("panic generating audit log"))
		}

		ret = retErr.ErrorOrNil()

		failure := float32(0.0)
		if ret != nil {
			failure = 1.0
		}
		metrics.IncrCounter([]string{"audit", "log_response_failure"}, failure)
	}()

	headers := in.Request.Headers
	defer func() {
		in.Request.Headers = headers
	}()

	// Ensure at least one backend logs
	anyLogged := false
	for name, be := range a.backends {
		in.Request.Headers = nil
		transHeaders, thErr := headersConfig.ApplyConfig(headers, be.backend.GetHash)
		if thErr != nil {
			a.logger.Error("audit: backend failed to include headers", "backend", name, "error", thErr)
			continue
		}
		in.Request.Headers = transHeaders

		start := time.Now()
		lrErr := be.backend.LogResponse(ctx, in)
		metrics.MeasureSince([]string{"audit", name, "log_response"}, start)
		if lrErr != nil {
			a.logger.Error("audit: backend failed to log response", "backend", name, "error", lrErr)
		} else {
			anyLogged = true
		}
	}
	if !anyLogged && len(a.backends) > 0 {
		retErr = multierror.Append(retErr, fmt.Errorf("no audit backend succeeded in logging the response"))
	}

	return retErr.ErrorOrNil()
}

func (a *AuditBroker) Invalidate(ctx context.Context, key string) {
	// For now we ignore the key as this would only apply to salts. We just
	// sort of brute force it on each one.
	a.Lock()
	defer a.Unlock()
	for _, be := range a.backends {
		be.backend.Invalidate(ctx)
	}
}
