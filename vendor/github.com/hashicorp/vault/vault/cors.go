package vault

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/vault/helper/strutil"
	"github.com/hashicorp/vault/logical"
)

const (
	CORSDisabled uint32 = iota
	CORSEnabled
)

var StdAllowedHeaders = []string{
	"Content-Type",
	"X-Requested-With",
	"X-Vault-AWS-IAM-Server-ID",
	"X-Vault-MFA",
	"X-Vault-No-Request-Forwarding",
	"X-Vault-Token",
	"X-Vault-Wrap-Format",
	"X-Vault-Wrap-TTL",
	"X-Vault-Policy-Override",
}

// CORSConfig stores the state of the CORS configuration.
type CORSConfig struct {
	sync.RWMutex   `json:"-"`
	core           *Core
	Enabled        uint32   `json:"enabled"`
	AllowedOrigins []string `json:"allowed_origins,omitempty"`
	AllowedHeaders []string `json:"allowed_headers,omitempty"`
}

func (c *Core) saveCORSConfig(ctx context.Context) error {
	view := c.systemBarrierView.SubView("config/")

	localConfig := &CORSConfig{
		Enabled: atomic.LoadUint32(&c.corsConfig.Enabled),
	}
	c.corsConfig.RLock()
	localConfig.AllowedOrigins = c.corsConfig.AllowedOrigins
	localConfig.AllowedHeaders = c.corsConfig.AllowedHeaders
	c.corsConfig.RUnlock()

	entry, err := logical.StorageEntryJSON("cors", localConfig)
	if err != nil {
		return fmt.Errorf("failed to create CORS config entry: %v", err)
	}

	if err := view.Put(ctx, entry); err != nil {
		return fmt.Errorf("failed to save CORS config: %v", err)
	}

	return nil
}

// This should only be called with the core state lock held for writing
func (c *Core) loadCORSConfig(ctx context.Context) error {
	view := c.systemBarrierView.SubView("config/")

	// Load the config in
	out, err := view.Get(ctx, "cors")
	if err != nil {
		return fmt.Errorf("failed to read CORS config: %v", err)
	}
	if out == nil {
		return nil
	}

	newConfig := new(CORSConfig)
	err = out.DecodeJSON(newConfig)
	if err != nil {
		return err
	}
	newConfig.core = c

	c.corsConfig = newConfig

	return nil
}

// Enable takes either a '*' or a comma-seprated list of URLs that can make
// cross-origin requests to Vault.
func (c *CORSConfig) Enable(ctx context.Context, urls []string, headers []string) error {
	if len(urls) == 0 {
		return errors.New("at least one origin or the wildcard must be provided.")
	}

	if strutil.StrListContains(urls, "*") && len(urls) > 1 {
		return errors.New("to allow all origins the '*' must be the only value for allowed_origins")
	}

	c.Lock()
	c.AllowedOrigins = urls

	// Start with the standard headers to Vault accepts.
	c.AllowedHeaders = append(c.AllowedHeaders, StdAllowedHeaders...)

	// Allow the user to add additional headers to the list of
	// headers allowed on cross-origin requests.
	if len(headers) > 0 {
		c.AllowedHeaders = append(c.AllowedHeaders, headers...)
	}
	c.Unlock()

	atomic.StoreUint32(&c.Enabled, CORSEnabled)

	return c.core.saveCORSConfig(ctx)
}

// IsEnabled returns the value of CORSConfig.isEnabled
func (c *CORSConfig) IsEnabled() bool {
	return atomic.LoadUint32(&c.Enabled) == CORSEnabled
}

// Disable sets CORS to disabled and clears the allowed origins & headers.
func (c *CORSConfig) Disable(ctx context.Context) error {
	atomic.StoreUint32(&c.Enabled, CORSDisabled)
	c.Lock()

	c.AllowedOrigins = nil
	c.AllowedHeaders = nil

	c.Unlock()

	return c.core.saveCORSConfig(ctx)
}

// IsValidOrigin determines if the origin of the request is allowed to make
// cross-origin requests based on the CORSConfig.
func (c *CORSConfig) IsValidOrigin(origin string) bool {
	// If we aren't enabling CORS then all origins are valid
	if !c.IsEnabled() {
		return true
	}

	c.RLock()
	defer c.RUnlock()

	if len(c.AllowedOrigins) == 0 {
		return false
	}

	if len(c.AllowedOrigins) == 1 && (c.AllowedOrigins)[0] == "*" {
		return true
	}

	return strutil.StrListContains(c.AllowedOrigins, origin)
}
