package http

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/vault"
)

type PrepareRequestFunc func(*vault.Core, *logical.Request) error

func buildLogicalRequest(core *vault.Core, w http.ResponseWriter, r *http.Request) (*logical.Request, int, error) {
	// Determine the path...
	if !strings.HasPrefix(r.URL.Path, "/v1/") {
		return nil, http.StatusNotFound, nil
	}
	path := r.URL.Path[len("/v1/"):]
	if path == "" {
		return nil, http.StatusNotFound, nil
	}

	// Determine the operation
	var op logical.Operation
	switch r.Method {
	case "DELETE":
		op = logical.DeleteOperation
	case "GET":
		op = logical.ReadOperation
		// Need to call ParseForm to get query params loaded
		queryVals := r.URL.Query()
		listStr := queryVals.Get("list")
		if listStr != "" {
			list, err := strconv.ParseBool(listStr)
			if err != nil {
				return nil, http.StatusBadRequest, nil
			}
			if list {
				op = logical.ListOperation
			}
		}
	case "POST", "PUT":
		op = logical.UpdateOperation
	case "LIST":
		op = logical.ListOperation
	case "OPTIONS":
	default:
		return nil, http.StatusMethodNotAllowed, nil
	}

	if op == logical.ListOperation {
		if !strings.HasSuffix(path, "/") {
			path += "/"
		}
	}

	// Parse the request if we can
	var data map[string]interface{}
	if op == logical.UpdateOperation {
		err := parseRequest(r, w, &data)
		if err == io.EOF {
			data = nil
			err = nil
		}
		if err != nil {
			return nil, http.StatusBadRequest, err
		}
	}

	// If we are a read operation, try and parse any parameters
	if op == logical.ReadOperation {
		getData := map[string]interface{}{}

		for k, v := range r.URL.Query() {
			// Skip the help key as this is a reserved parameter
			if k == "help" {
				continue
			}

			switch {
			case len(v) == 0:
			case len(v) == 1:
				getData[k] = v[0]
			default:
				getData[k] = v
			}
		}

		if len(getData) > 0 {
			data = getData
		}
	}

	var err error
	request_id, err := uuid.GenerateUUID()
	if err != nil {
		return nil, http.StatusBadRequest, errwrap.Wrapf("failed to generate identifier for the request: {{err}}", err)
	}

	req := requestAuth(core, r, &logical.Request{
		ID:         request_id,
		Operation:  op,
		Path:       path,
		Data:       data,
		Connection: getConnection(r),
		Headers:    r.Header,
	})

	req, err = requestWrapInfo(r, req)
	if err != nil {
		return nil, http.StatusBadRequest, errwrap.Wrapf("error parsing X-Vault-Wrap-TTL header: {{err}}", err)
	}

	return req, 0, nil
}

func handleLogical(core *vault.Core, injectDataIntoTopLevel bool, prepareRequestCallback PrepareRequestFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, statusCode, err := buildLogicalRequest(core, w, r)
		if err != nil || statusCode != 0 {
			respondError(w, statusCode, err)
			return
		}

		// Certain endpoints may require changes to the request object. They
		// will have a callback registered to do the needed operations, so
		// invoke it before proceeding.
		if prepareRequestCallback != nil {
			if err := prepareRequestCallback(core, req); err != nil {
				respondError(w, http.StatusBadRequest, err)
				return
			}
		}

		// Make the internal request. We attach the connection info
		// as well in case this is an authentication request that requires
		// it. Vault core handles stripping this if we need to. This also
		// handles all error cases; if we hit respondLogical, the request is a
		// success.
		resp, ok := request(core, w, r, req)
		if !ok {
			return
		}

		// Build the proper response
		respondLogical(w, r, req, injectDataIntoTopLevel, resp)
	})
}

func respondLogical(w http.ResponseWriter, r *http.Request, req *logical.Request, injectDataIntoTopLevel bool, resp *logical.Response) {
	var httpResp *logical.HTTPResponse
	var ret interface{}

	if resp != nil {
		if resp.Redirect != "" {
			// If we have a redirect, redirect! We use a 307 code
			// because we don't actually know if its permanent.
			http.Redirect(w, r, resp.Redirect, 307)
			return
		}

		// Check if this is a raw response
		if _, ok := resp.Data[logical.HTTPStatusCode]; ok {
			respondRaw(w, r, resp)
			return
		}

		if resp.WrapInfo != nil && resp.WrapInfo.Token != "" {
			httpResp = &logical.HTTPResponse{
				WrapInfo: &logical.HTTPWrapInfo{
					Token:           resp.WrapInfo.Token,
					Accessor:        resp.WrapInfo.Accessor,
					TTL:             int(resp.WrapInfo.TTL.Seconds()),
					CreationTime:    resp.WrapInfo.CreationTime.Format(time.RFC3339Nano),
					CreationPath:    resp.WrapInfo.CreationPath,
					WrappedAccessor: resp.WrapInfo.WrappedAccessor,
				},
			}
		} else {
			httpResp = logical.LogicalResponseToHTTPResponse(resp)
			httpResp.RequestID = req.ID
		}

		ret = httpResp

		if injectDataIntoTopLevel {
			injector := logical.HTTPSysInjector{
				Response: httpResp,
			}
			ret = injector
		}
	}

	// Respond
	respondOk(w, ret)
	return
}

// respondRaw is used when the response is using HTTPContentType and HTTPRawBody
// to change the default response handling. This is only used for specific things like
// returning the CRL information on the PKI backends.
func respondRaw(w http.ResponseWriter, r *http.Request, resp *logical.Response) {
	retErr := func(w http.ResponseWriter, err string) {
		w.Header().Set("X-Vault-Raw-Error", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(nil)
	}

	// Ensure this is never a secret or auth response
	if resp.Secret != nil || resp.Auth != nil {
		retErr(w, "raw responses cannot contain secrets or auth")
		return
	}

	// Get the status code
	statusRaw, ok := resp.Data[logical.HTTPStatusCode]
	if !ok {
		retErr(w, "no status code given")
		return
	}

	var status int
	switch statusRaw.(type) {
	case int:
		status = statusRaw.(int)
	case float64:
		status = int(statusRaw.(float64))
	case json.Number:
		s64, err := statusRaw.(json.Number).Float64()
		if err != nil {
			retErr(w, "cannot decode status code")
			return
		}
		status = int(s64)
	default:
		retErr(w, "cannot decode status code")
		return
	}

	nonEmpty := status != http.StatusNoContent

	var contentType string
	var body []byte

	// Get the content type header; don't require it if the body is empty
	contentTypeRaw, ok := resp.Data[logical.HTTPContentType]
	if !ok && nonEmpty {
		retErr(w, "no content type given")
		return
	}
	if ok {
		contentType, ok = contentTypeRaw.(string)
		if !ok {
			retErr(w, "cannot decode content type")
			return
		}
	}

	if nonEmpty {
		// Get the body
		bodyRaw, ok := resp.Data[logical.HTTPRawBody]
		if !ok {
			retErr(w, "no body given")
			return
		}

		switch bodyRaw.(type) {
		case string:
			var err error
			body, err = base64.StdEncoding.DecodeString(bodyRaw.(string))
			if err != nil {
				retErr(w, "cannot decode body")
				return
			}
		case []byte:
			body = bodyRaw.([]byte)
		default:
			retErr(w, "cannot decode body")
			return
		}
	}

	// Write the response
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	w.WriteHeader(status)
	w.Write(body)
}

// getConnection is used to format the connection information for
// attaching to a logical request
func getConnection(r *http.Request) (connection *logical.Connection) {
	var remoteAddr string

	remoteAddr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteAddr = ""
	}

	connection = &logical.Connection{
		RemoteAddr: remoteAddr,
		ConnState:  r.TLS,
	}
	return
}
