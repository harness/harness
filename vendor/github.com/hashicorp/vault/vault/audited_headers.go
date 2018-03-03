package vault

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/vault/logical"
)

// N.B.: While we could use textproto to get the canonical mime header, HTTP/2
// requires all headers to be converted to lower case, so we just do that.

const (
	// Key used in the BarrierView to store and retrieve the header config
	auditedHeadersEntry = "audited-headers"
	// Path used to create a sub view off of BarrierView
	auditedHeadersSubPath = "audited-headers-config/"
)

type auditedHeaderSettings struct {
	HMAC bool `json:"hmac"`
}

// AuditedHeadersConfig is used by the Audit Broker to write only approved
// headers to the audit logs. It uses a BarrierView to persist the settings.
type AuditedHeadersConfig struct {
	Headers map[string]*auditedHeaderSettings

	view *BarrierView
	sync.RWMutex
}

// add adds or overwrites a header in the config and updates the barrier view
func (a *AuditedHeadersConfig) add(ctx context.Context, header string, hmac bool) error {
	if header == "" {
		return fmt.Errorf("header value cannot be empty")
	}

	// Grab a write lock
	a.Lock()
	defer a.Unlock()

	if a.Headers == nil {
		a.Headers = make(map[string]*auditedHeaderSettings, 1)
	}

	a.Headers[strings.ToLower(header)] = &auditedHeaderSettings{hmac}
	entry, err := logical.StorageEntryJSON(auditedHeadersEntry, a.Headers)
	if err != nil {
		return fmt.Errorf("failed to persist audited headers config: %v", err)
	}

	if err := a.view.Put(ctx, entry); err != nil {
		return fmt.Errorf("failed to persist audited headers config: %v", err)
	}

	return nil
}

// remove deletes a header out of the header config and updates the barrier view
func (a *AuditedHeadersConfig) remove(ctx context.Context, header string) error {
	if header == "" {
		return fmt.Errorf("header value cannot be empty")
	}

	// Grab a write lock
	a.Lock()
	defer a.Unlock()

	// Nothing to delete
	if len(a.Headers) == 0 {
		return nil
	}

	delete(a.Headers, strings.ToLower(header))
	entry, err := logical.StorageEntryJSON(auditedHeadersEntry, a.Headers)
	if err != nil {
		return fmt.Errorf("failed to persist audited headers config: %v", err)
	}

	if err := a.view.Put(ctx, entry); err != nil {
		return fmt.Errorf("failed to persist audited headers config: %v", err)
	}

	return nil
}

// ApplyConfig returns a map of approved headers and their values, either
// hmac'ed or plaintext
func (a *AuditedHeadersConfig) ApplyConfig(headers map[string][]string, hashFunc func(string) (string, error)) (result map[string][]string, retErr error) {
	// Grab a read lock
	a.RLock()
	defer a.RUnlock()

	// Make a copy of the incoming headers with everything lower so we can
	// case-insensitively compare
	lowerHeaders := make(map[string][]string, len(headers))
	for k, v := range headers {
		lowerHeaders[strings.ToLower(k)] = v
	}

	result = make(map[string][]string, len(a.Headers))
	for key, settings := range a.Headers {
		if val, ok := lowerHeaders[key]; ok {
			// copy the header values so we don't overwrite them
			hVals := make([]string, len(val))
			copy(hVals, val)

			// Optionally hmac the values
			if settings.HMAC {
				for i, el := range hVals {
					hVal, err := hashFunc(el)
					if err != nil {
						return nil, err
					}
					hVals[i] = hVal
				}
			}

			result[key] = hVals
		}
	}

	return result, nil
}

// Initalize the headers config by loading from the barrier view
func (c *Core) setupAuditedHeadersConfig(ctx context.Context) error {
	// Create a sub-view
	view := c.systemBarrierView.SubView(auditedHeadersSubPath)

	// Create the config
	out, err := view.Get(ctx, auditedHeadersEntry)
	if err != nil {
		return fmt.Errorf("failed to read config: %v", err)
	}

	headers := make(map[string]*auditedHeaderSettings)
	if out != nil {
		err = out.DecodeJSON(&headers)
		if err != nil {
			return err
		}
	}

	// Ensure that we are able to case-sensitively access the headers;
	// necessary for the upgrade case
	lowerHeaders := make(map[string]*auditedHeaderSettings, len(headers))
	for k, v := range headers {
		lowerHeaders[strings.ToLower(k)] = v
	}

	c.auditedHeaders = &AuditedHeadersConfig{
		Headers: lowerHeaders,
		view:    view,
	}

	return nil
}
