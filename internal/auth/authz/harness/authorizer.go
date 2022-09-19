// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package harness

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

var _ authz.Authorizer = (*Authorizer)(nil)

type Authorizer struct {
	client      *http.Client
	aclEndpoint string
	authToken   string
}

func NewAuthorizer(aclEndpoint, authToken string) (authz.Authorizer, error) {
	// build http client - could be injected, too
	tr := &http.Transport{
		// TODO: expose InsecureSkipVerify in config
		TLSClientConfig: &tls.Config{
			//nolint:gosec // accept any host cert
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{Transport: tr}

	return &Authorizer{
		client:      client,
		aclEndpoint: aclEndpoint,
		authToken:   authToken,
	}, nil
}

func (a *Authorizer) Check(ctx context.Context, principalType enum.PrincipalType, principalID string,
	scope *types.Scope, resource *types.Resource, permission enum.Permission) (bool, error) {
	return a.CheckAll(ctx, principalType, principalID, types.PermissionCheck{
		Scope:      *scope,
		Resource:   *resource,
		Permission: permission,
	})
}

func (a *Authorizer) CheckAll(ctx context.Context, principalType enum.PrincipalType, principalID string,
	permissionChecks ...types.PermissionCheck) (bool, error) {
	if len(permissionChecks) == 0 {
		return false, authz.ErrNoPermissionCheckProvided
	}

	requestDto, err := createACLRequest(principalType, principalID, permissionChecks)
	if err != nil {
		return false, err
	}
	byt, err := json.Marshal(requestDto)
	if err != nil {
		return false, err
	}

	// TODO: accountId might be different!
	url := a.aclEndpoint + "?routingId=" + requestDto.Permissions[0].ResourceScope.AccountIdentifier
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(byt))
	if err != nil {
		return false, err
	}

	httpRequest.Header = http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{"Bearer " + a.authToken},
	}

	response, err := a.client.Do(httpRequest)
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		return false, err
	}

	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("got unexpected status code '%d' - assume unauthorized", response.StatusCode)
	}

	bodyByte, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return false, err
	}

	var responseDto aclResponse
	err = json.Unmarshal(bodyByte, &responseDto)
	if err != nil {
		return false, err
	}

	return checkACLResponse(permissionChecks, responseDto)
}

func createACLRequest(principalType enum.PrincipalType, principalID string,
	permissionChecks []types.PermissionCheck) (*aclRequest, error) {
	// Generate ACL req
	req := aclRequest{
		Permissions: []aclPermission{},
		Principal: aclPrincipal{
			PrincipalIdentifier: principalID,
			PrincipalType:       string(principalType),
		},
	}

	// map all permissionchecks to ACL permission checks
	for _, c := range permissionChecks {
		mappedPermission := mapPermission(c.Permission)
		mappedResourceScope, err := mapScope(c.Scope)
		if err != nil {
			return nil, err
		}

		req.Permissions = append(req.Permissions, aclPermission{
			Permission:         mappedPermission,
			ResourceScope:      *mappedResourceScope,
			ResourceType:       string(c.Resource.Type),
			ResourceIdentifier: c.Resource.Name,
		})
	}

	return &req, nil
}

func checkACLResponse(permissionChecks []types.PermissionCheck, responseDto aclResponse) (bool, error) {
	/*
	 * We are assuming two things:
	 *  - All permission checks were made for the same principal.
	 *  - Permissions inherit down the hierarchy (Account -> Organization -> Project -> Repository)
	 *	- No two checks are for the same permission - is similar to ff implementation:
	 *		https://github.com/wings-software/ff-server/blob/master/pkg/rbac/client.go#L88
	 *
	 * Based on that, if there's any permitted result for a permission check the permission is allowed.
	 * Now we just have to ensure that all permissions are allowed
	 *
	 * TODO: Use resource name + scope for verifying results.
	 */

	for _, check := range permissionChecks {
		permissionPermitted := false
		for _, ace := range responseDto.Data.AccessControlList {
			if string(check.Permission) == ace.Permission && ace.Permitted {
				permissionPermitted = true
				break
			}
		}

		if !permissionPermitted {
			return false, fmt.Errorf("permission '%s' is not permitted according to ACL (correlationId: '%s')",
				check.Permission,
				responseDto.CorrelationID)
		}
	}

	return true, nil
}

func mapScope(scope types.Scope) (*aclResourceScope, error) {
	/*
	 * ASSUMPTION:
	 *	Harness embeded structure is mapped to the following scm space:
	 *      {Account}/{Organization}/{Project}
	 *
	 * We can use that assumption to translate back from scope.spacePath to harness scope.
	 * However, this only works as long as resources exist within spaces only.
	 * For controlling access to any child resources of a repository, harness doesn't have a matching
	 * structure out of the box (e.g. branches, ...)
	 *
	 * IMPORTANT:
	 * 		For now harness embedded doesn't support scope.Repository (has to be configured on space level ...)
	 *
	 * TODO: Handle scope.Repository in harness embedded mode
	 */

	const (
		accIndex     = 0
		orgIndex     = 1
		projectIndex = 2
		scopes       = 3
	)

	harnessIdentifiers := strings.Split(scope.SpacePath, "/")
	if len(harnessIdentifiers) > scopes {
		return nil, fmt.Errorf("unable to convert '%s' to harness resource scope "+
			"(expected {Account}/{Organization}/{Project} or a sub scope)", scope.SpacePath)
	}

	aclScope := &aclResourceScope{}
	if len(harnessIdentifiers) > accIndex {
		aclScope.AccountIdentifier = harnessIdentifiers[accIndex]
	}
	if len(harnessIdentifiers) > orgIndex {
		aclScope.OrgIdentifier = harnessIdentifiers[orgIndex]
	}
	if len(harnessIdentifiers) > projectIndex {
		aclScope.ProjectIdentifier = harnessIdentifiers[projectIndex]
	}

	return aclScope, nil
}

func mapPermission(permission enum.Permission) string {
	// harness has multiple modules - add scm prefix
	return "scm_" + string(permission)
}
