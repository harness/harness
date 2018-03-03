package vault

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/armon/go-metrics"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/vault/audit"
	"github.com/hashicorp/vault/helper/consts"
	"github.com/hashicorp/vault/helper/identity"
	"github.com/hashicorp/vault/helper/jsonutil"
	"github.com/hashicorp/vault/helper/policyutil"
	"github.com/hashicorp/vault/helper/strutil"
	"github.com/hashicorp/vault/helper/wrapping"
	"github.com/hashicorp/vault/logical"
)

const (
	replTimeout = 10 * time.Second
)

// HandleRequest is used to handle a new incoming request
func (c *Core) HandleRequest(req *logical.Request) (resp *logical.Response, err error) {
	c.stateLock.RLock()
	defer c.stateLock.RUnlock()
	if c.sealed {
		return nil, consts.ErrSealed
	}
	if c.standby {
		return nil, consts.ErrStandby
	}

	ctx, cancel := context.WithCancel(c.activeContext)
	defer cancel()

	// Allowing writing to a path ending in / makes it extremely difficult to
	// understand user intent for the filesystem-like backends (kv,
	// cubbyhole) -- did they want a key named foo/ or did they want to write
	// to a directory foo/ with no (or forgotten) key, or...? It also affects
	// lookup, because paths ending in / are considered prefixes by some
	// backends. Basically, it's all just terrible, so don't allow it.
	if strings.HasSuffix(req.Path, "/") &&
		(req.Operation == logical.UpdateOperation ||
			req.Operation == logical.CreateOperation) {
		return logical.ErrorResponse("cannot write to a path ending in '/'"), nil
	}

	var auth *logical.Auth
	if c.router.LoginPath(req.Path) {
		resp, auth, err = c.handleLoginRequest(ctx, req)
	} else {
		resp, auth, err = c.handleRequest(ctx, req)
	}

	// Ensure we don't leak internal data
	if resp != nil {
		if resp.Secret != nil {
			resp.Secret.InternalData = nil
		}
		if resp.Auth != nil {
			resp.Auth.InternalData = nil
		}
	}

	// We are wrapping if there is anything to wrap (not a nil response) and a
	// TTL was specified for the token. Errors on a call should be returned to
	// the caller, so wrapping is turned off if an error is hit and the error
	// is logged to the audit log.
	wrapping := resp != nil &&
		err == nil &&
		!resp.IsError() &&
		resp.WrapInfo != nil &&
		resp.WrapInfo.TTL != 0 &&
		resp.WrapInfo.Token == ""

	if wrapping {
		cubbyResp, cubbyErr := c.wrapInCubbyhole(ctx, req, resp, auth)
		// If not successful, returns either an error response from the
		// cubbyhole backend or an error; if either is set, set resp and err to
		// those and continue so that that's what we audit log. Otherwise
		// finish the wrapping and audit log that.
		if cubbyResp != nil || cubbyErr != nil {
			resp = cubbyResp
			err = cubbyErr
		} else {
			wrappingResp := &logical.Response{
				WrapInfo: resp.WrapInfo,
				Warnings: resp.Warnings,
			}
			resp = wrappingResp
		}
	}

	auditResp := resp
	// When unwrapping we want to log the actual response that will be written
	// out. We still want to return the raw value to avoid automatic updating
	// to any of it.
	if req.Path == "sys/wrapping/unwrap" &&
		resp != nil &&
		resp.Data != nil &&
		resp.Data[logical.HTTPRawBody] != nil {

		// Decode the JSON
		httpResp := &logical.HTTPResponse{}
		err := jsonutil.DecodeJSON(resp.Data[logical.HTTPRawBody].([]byte), httpResp)
		if err != nil {
			c.logger.Error("core: failed to unmarshal wrapped HTTP response for audit logging", "error", err)
			return nil, ErrInternalError
		}

		auditResp = logical.HTTPResponseToLogicalResponse(httpResp)
	}

	var nonHMACReqDataKeys []string
	var nonHMACRespDataKeys []string
	entry := c.router.MatchingMountEntry(req.Path)
	if entry != nil {
		// Get and set ignored HMAC'd value. Reset those back to empty afterwards.
		if rawVals, ok := entry.synthesizedConfigCache.Load("audit_non_hmac_request_keys"); ok {
			nonHMACReqDataKeys = rawVals.([]string)
		}

		// Get and set ignored HMAC'd value. Reset those back to empty afterwards.
		if auditResp != nil {
			if rawVals, ok := entry.synthesizedConfigCache.Load("audit_non_hmac_response_keys"); ok {
				nonHMACRespDataKeys = rawVals.([]string)
			}
		}
	}

	// Create an audit trail of the response
	logInput := &audit.LogInput{
		Auth:                auth,
		Request:             req,
		Response:            auditResp,
		OuterErr:            err,
		NonHMACReqDataKeys:  nonHMACReqDataKeys,
		NonHMACRespDataKeys: nonHMACRespDataKeys,
	}
	if auditErr := c.auditBroker.LogResponse(ctx, logInput, c.auditedHeaders); auditErr != nil {
		c.logger.Error("core: failed to audit response", "request_path", req.Path, "error", auditErr)
		return nil, ErrInternalError
	}

	return
}

func (c *Core) handleRequest(ctx context.Context, req *logical.Request) (retResp *logical.Response, retAuth *logical.Auth, retErr error) {
	defer metrics.MeasureSince([]string{"core", "handle_request"}, time.Now())

	var nonHMACReqDataKeys []string
	entry := c.router.MatchingMountEntry(req.Path)
	if entry != nil {
		// Get and set ignored HMAC'd value.
		if rawVals, ok := entry.synthesizedConfigCache.Load("audit_non_hmac_request_keys"); ok {
			nonHMACReqDataKeys = rawVals.([]string)
		}
	}

	// Validate the token
	auth, te, ctErr := c.checkToken(ctx, req, false)
	// We run this logic first because we want to decrement the use count even in the case of an error
	if te != nil {
		// Attempt to use the token (decrement NumUses)
		var err error
		te, err = c.tokenStore.UseToken(ctx, te)
		if err != nil {
			c.logger.Error("core: failed to use token", "error", err)
			retErr = multierror.Append(retErr, ErrInternalError)
			return nil, nil, retErr
		}
		if te == nil {
			// Token has been revoked by this point
			retErr = multierror.Append(retErr, logical.ErrPermissionDenied)
			return nil, nil, retErr
		}
		if te.NumUses == -1 {
			// We defer a revocation until after logic has run, since this is a
			// valid request (this is the token's final use). We pass the ID in
			// directly just to be safe in case something else modifies te later.
			defer func(id string) {
				err = c.tokenStore.Revoke(ctx, id)
				if err != nil {
					c.logger.Error("core: failed to revoke token", "error", err)
					retResp = nil
					retAuth = nil
					retErr = multierror.Append(retErr, ErrInternalError)
				}
				if retResp != nil && retResp.Secret != nil &&
					// Some backends return a TTL even without a Lease ID
					retResp.Secret.LeaseID != "" {
					retResp = logical.ErrorResponse("Secret cannot be returned; token had one use left, so leased credentials were immediately revoked.")
					return
				}
			}(te.ID)
		}
	}
	if ctErr != nil {
		// If it is an internal error we return that, otherwise we
		// return invalid request so that the status codes can be correct
		errType := logical.ErrInvalidRequest
		switch ctErr {
		case ErrInternalError, logical.ErrPermissionDenied:
			errType = ctErr
		}

		logInput := &audit.LogInput{
			Auth:               auth,
			Request:            req,
			OuterErr:           ctErr,
			NonHMACReqDataKeys: nonHMACReqDataKeys,
		}
		if err := c.auditBroker.LogRequest(ctx, logInput, c.auditedHeaders); err != nil {
			c.logger.Error("core: failed to audit request", "path", req.Path, "error", err)
		}

		if errType != nil {
			retErr = multierror.Append(retErr, errType)
		}
		if ctErr == ErrInternalError {
			return nil, auth, retErr
		}
		return logical.ErrorResponse(ctErr.Error()), auth, retErr
	}

	// Attach the display name
	req.DisplayName = auth.DisplayName

	// Create an audit trail of the request
	logInput := &audit.LogInput{
		Auth:               auth,
		Request:            req,
		NonHMACReqDataKeys: nonHMACReqDataKeys,
	}
	if err := c.auditBroker.LogRequest(ctx, logInput, c.auditedHeaders); err != nil {
		c.logger.Error("core: failed to audit request", "path", req.Path, "error", err)
		retErr = multierror.Append(retErr, ErrInternalError)
		return nil, auth, retErr
	}

	// Route the request
	resp, routeErr := c.router.Route(ctx, req)
	if resp != nil {
		// If wrapping is used, use the shortest between the request and response
		var wrapTTL time.Duration
		var wrapFormat, creationPath string
		var sealWrap bool

		// Ensure no wrap info information is set other than, possibly, the TTL
		if resp.WrapInfo != nil {
			if resp.WrapInfo.TTL > 0 {
				wrapTTL = resp.WrapInfo.TTL
			}
			wrapFormat = resp.WrapInfo.Format
			creationPath = resp.WrapInfo.CreationPath
			sealWrap = resp.WrapInfo.SealWrap
			resp.WrapInfo = nil
		}

		if req.WrapInfo != nil {
			if req.WrapInfo.TTL > 0 {
				switch {
				case wrapTTL == 0:
					wrapTTL = req.WrapInfo.TTL
				case req.WrapInfo.TTL < wrapTTL:
					wrapTTL = req.WrapInfo.TTL
				}
			}
			// If the wrap format hasn't been set by the response, set it to
			// the request format
			if req.WrapInfo.Format != "" && wrapFormat == "" {
				wrapFormat = req.WrapInfo.Format
			}
		}

		if wrapTTL > 0 {
			resp.WrapInfo = &wrapping.ResponseWrapInfo{
				TTL:          wrapTTL,
				Format:       wrapFormat,
				CreationPath: creationPath,
				SealWrap:     sealWrap,
			}
		}
	}

	// If there is a secret, we must register it with the expiration manager.
	// We exclude renewal of a lease, since it does not need to be re-registered
	if resp != nil && resp.Secret != nil && !strings.HasPrefix(req.Path, "sys/renew") &&
		!strings.HasPrefix(req.Path, "sys/leases/renew") {
		// Get the SystemView for the mount
		sysView := c.router.MatchingSystemView(req.Path)
		if sysView == nil {
			c.logger.Error("core: unable to retrieve system view from router")
			retErr = multierror.Append(retErr, ErrInternalError)
			return nil, auth, retErr
		}

		// Apply the default lease if none given
		if resp.Secret.TTL == 0 {
			resp.Secret.TTL = sysView.DefaultLeaseTTL()
		}

		// Limit the lease duration
		maxTTL := sysView.MaxLeaseTTL()
		if resp.Secret.TTL > maxTTL {
			resp.Secret.TTL = maxTTL
		}

		// KV mounts should return the TTL but not register
		// for a lease as this provides a massive slowdown
		registerLease := true
		matchingBackend := c.router.MatchingBackend(req.Path)
		if matchingBackend == nil {
			c.logger.Error("core: unable to retrieve kv backend from router")
			retErr = multierror.Append(retErr, ErrInternalError)
			return nil, auth, retErr
		}
		if ptbe, ok := matchingBackend.(*PassthroughBackend); ok {
			if !ptbe.GeneratesLeases() {
				registerLease = false
				resp.Secret.Renewable = false
			}
		}

		if registerLease {
			leaseID, err := c.expiration.Register(req, resp)
			if err != nil {
				c.logger.Error("core: failed to register lease", "request_path", req.Path, "error", err)
				retErr = multierror.Append(retErr, ErrInternalError)
				return nil, auth, retErr
			}
			resp.Secret.LeaseID = leaseID
		}
	}

	// If the request was to renew a token, and if there are group aliases set
	// in the auth object, then the group memberships should be refreshed
	if strings.HasPrefix(req.Path, "auth/token/renew") &&
		resp != nil &&
		resp.Auth != nil &&
		resp.Auth.EntityID != "" &&
		resp.Auth.GroupAliases != nil {
		err := c.identityStore.refreshExternalGroupMembershipsByEntityID(resp.Auth.EntityID, resp.Auth.GroupAliases)
		if err != nil {
			c.logger.Error("core: failed to refresh external group memberships", "error", err)
			retErr = multierror.Append(retErr, ErrInternalError)
			return nil, auth, retErr
		}
	}

	// Only the token store is allowed to return an auth block, for any
	// other request this is an internal error. We exclude renewal of a token,
	// since it does not need to be re-registered
	if resp != nil && resp.Auth != nil && !strings.HasPrefix(req.Path, "auth/token/renew") {
		if !strings.HasPrefix(req.Path, "auth/token/") {
			c.logger.Error("core: unexpected Auth response for non-token backend", "request_path", req.Path)
			retErr = multierror.Append(retErr, ErrInternalError)
			return nil, auth, retErr
		}

		// Register with the expiration manager. We use the token's actual path
		// here because roles allow suffixes.
		te, err := c.tokenStore.Lookup(ctx, resp.Auth.ClientToken)
		if err != nil {
			c.logger.Error("core: failed to look up token", "error", err)
			retErr = multierror.Append(retErr, ErrInternalError)
			return nil, auth, retErr
		}

		if err := c.expiration.RegisterAuth(te.Path, resp.Auth); err != nil {
			c.tokenStore.Revoke(ctx, te.ID)
			c.logger.Error("core: failed to register token lease", "request_path", req.Path, "error", err)
			retErr = multierror.Append(retErr, ErrInternalError)
			return nil, auth, retErr
		}
	}

	if resp != nil &&
		req.Path == "cubbyhole/response" &&
		len(te.Policies) == 1 &&
		te.Policies[0] == responseWrappingPolicyName {
		resp.AddWarning("Reading from 'cubbyhole/response' is deprecated. Please use sys/wrapping/unwrap to unwrap responses, as it provides additional security checks and other benefits.")
	}

	// Return the response and error
	if routeErr != nil {
		retErr = multierror.Append(retErr, routeErr)
	}

	return resp, auth, retErr
}

// handleLoginRequest is used to handle a login request, which is an
// unauthenticated request to the backend.
func (c *Core) handleLoginRequest(ctx context.Context, req *logical.Request) (retResp *logical.Response, retAuth *logical.Auth, retErr error) {
	defer metrics.MeasureSince([]string{"core", "handle_login_request"}, time.Now())

	req.Unauthenticated = true

	var auth *logical.Auth
	// Create an audit trail of the request, auth is not available on login requests
	// Create an audit trail of the request. Attach auth if it was returned,
	// e.g. if a token was provided.
	logInput := &audit.LogInput{
		Auth:    auth,
		Request: req,
	}
	if err := c.auditBroker.LogRequest(ctx, logInput, c.auditedHeaders); err != nil {
		c.logger.Error("core: failed to audit request", "path", req.Path, "error", err)
		return nil, nil, ErrInternalError
	}

	// The token store uses authentication even when creating a new token,
	// so it's handled in handleRequest. It should not be reached here.
	if strings.HasPrefix(req.Path, "auth/token/") {
		c.logger.Error("core: unexpected login request for token backend", "request_path", req.Path)
		return nil, nil, ErrInternalError
	}

	// Route the request
	resp, routeErr := c.router.Route(ctx, req)
	if resp != nil {
		// If wrapping is used, use the shortest between the request and response
		var wrapTTL time.Duration
		var wrapFormat, creationPath string
		var sealWrap bool

		// Ensure no wrap info information is set other than, possibly, the TTL
		if resp.WrapInfo != nil {
			if resp.WrapInfo.TTL > 0 {
				wrapTTL = resp.WrapInfo.TTL
			}
			wrapFormat = resp.WrapInfo.Format
			creationPath = resp.WrapInfo.CreationPath
			sealWrap = resp.WrapInfo.SealWrap
			resp.WrapInfo = nil
		}

		if req.WrapInfo != nil {
			if req.WrapInfo.TTL > 0 {
				switch {
				case wrapTTL == 0:
					wrapTTL = req.WrapInfo.TTL
				case req.WrapInfo.TTL < wrapTTL:
					wrapTTL = req.WrapInfo.TTL
				}
			}
			if req.WrapInfo.Format != "" && wrapFormat == "" {
				wrapFormat = req.WrapInfo.Format
			}
		}

		if wrapTTL > 0 {
			resp.WrapInfo = &wrapping.ResponseWrapInfo{
				TTL:          wrapTTL,
				Format:       wrapFormat,
				CreationPath: creationPath,
				SealWrap:     sealWrap,
			}
		}
	}

	// A login request should never return a secret!
	if resp != nil && resp.Secret != nil {
		c.logger.Error("core: unexpected Secret response for login path", "request_path", req.Path)
		return nil, nil, ErrInternalError
	}

	// If the response generated an authentication, then generate the token
	if resp != nil && resp.Auth != nil {
		var entity *identity.Entity
		auth = resp.Auth

		if auth.Alias != nil {
			// Overwrite the mount type and mount path in the alias
			// information
			auth.Alias.MountType = req.MountType
			auth.Alias.MountAccessor = req.MountAccessor

			if auth.Alias.Name == "" {
				return nil, nil, fmt.Errorf("missing name in alias")
			}

			var err error

			// Fetch the entity for the alias, or create an entity if one
			// doesn't exist.
			entity, err = c.identityStore.CreateOrFetchEntity(auth.Alias)
			if err != nil {
				return nil, nil, err
			}

			if entity == nil {
				return nil, nil, fmt.Errorf("failed to create an entity for the authenticated alias")
			}

			auth.EntityID = entity.ID
			if auth.GroupAliases != nil {
				err = c.identityStore.refreshExternalGroupMembershipsByEntityID(auth.EntityID, auth.GroupAliases)
				if err != nil {
					return nil, nil, err
				}
			}
		}

		if strutil.StrListSubset(auth.Policies, []string{"root"}) {
			return logical.ErrorResponse("auth methods cannot create root tokens"), nil, logical.ErrInvalidRequest
		}

		// Determine the source of the login
		source := c.router.MatchingMount(req.Path)
		source = strings.TrimPrefix(source, credentialRoutePrefix)
		source = strings.Replace(source, "/", "-", -1)

		// Prepend the source to the display name
		auth.DisplayName = strings.TrimSuffix(source+auth.DisplayName, "-")

		sysView := c.router.MatchingSystemView(req.Path)
		if sysView == nil {
			c.logger.Error("core: unable to look up sys view for login path", "request_path", req.Path)
			return nil, nil, ErrInternalError
		}

		// Start off with the sys default value, and update according to period/TTL
		// from resp.Auth
		tokenTTL := sysView.DefaultLeaseTTL()

		switch {
		case auth.Period > time.Duration(0):
			// Cap the period value to the sys max_ttl value. The auth backend should
			// have checked for it on its login path, but we check here again for
			// sanity.
			if auth.Period > sysView.MaxLeaseTTL() {
				auth.Period = sysView.MaxLeaseTTL()
			}
			tokenTTL = auth.Period
		case auth.TTL > time.Duration(0):
			// Cap the TTL value. The auth backend should have checked for it on its
			// login path (e.g. a call to b.SanitizeTTL), but we check here again for
			// sanity.
			if auth.TTL > sysView.MaxLeaseTTL() {
				auth.TTL = sysView.MaxLeaseTTL()
			}
			tokenTTL = auth.TTL
		}

		// Generate a token
		te := TokenEntry{
			Path:         req.Path,
			Policies:     auth.Policies,
			Meta:         auth.Metadata,
			DisplayName:  auth.DisplayName,
			CreationTime: time.Now().Unix(),
			TTL:          tokenTTL,
			NumUses:      auth.NumUses,
			EntityID:     auth.EntityID,
		}

		te.Policies = policyutil.SanitizePolicies(te.Policies, true)

		// Prevent internal policies from being assigned to tokens
		for _, policy := range te.Policies {
			if strutil.StrListContains(nonAssignablePolicies, policy) {
				return logical.ErrorResponse(fmt.Sprintf("cannot assign policy %q", policy)), nil, logical.ErrInvalidRequest
			}
		}

		if err := c.tokenStore.create(ctx, &te); err != nil {
			c.logger.Error("core: failed to create token", "error", err)
			return nil, auth, ErrInternalError
		}

		// Populate the client token, accessor, and TTL
		auth.ClientToken = te.ID
		auth.Accessor = te.Accessor
		auth.Policies = te.Policies
		auth.TTL = te.TTL

		// Register with the expiration manager
		if err := c.expiration.RegisterAuth(te.Path, auth); err != nil {
			c.tokenStore.Revoke(ctx, te.ID)
			c.logger.Error("core: failed to register token lease", "request_path", req.Path, "error", err)
			return nil, auth, ErrInternalError
		}

		// Attach the display name, might be used by audit backends
		req.DisplayName = auth.DisplayName
	}

	return resp, auth, routeErr
}
