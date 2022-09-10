// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package harness

import (
	"bytes"
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
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	return &Authorizer{
		client:      client,
		aclEndpoint: aclEndpoint,
		authToken:   authToken,
	}, nil
}

func (a *Authorizer) Check(principalType enum.PrincipalType, principalId string, scope *types.Scope, resource *types.Resource, permission enum.Permission) (bool, error) {
	return a.CheckAll(principalType, principalId, &types.PermissionCheck{Scope: *scope, Resource: *resource, Permission: permission})
}

func (a *Authorizer) CheckAll(principalType enum.PrincipalType, principalId string, permissionChecks ...*types.PermissionCheck) (bool, error) {
	if len(permissionChecks) == 0 {
		return false, authz.ErrNoPermissionCheckProvided
	}

	requestDto, err := createAclRequest(principalType, principalId, permissionChecks)
	byt, err := json.Marshal(requestDto)
	if err != nil {
		return false, err
	}

	// TODO: accountId might be different!
	url := a.aclEndpoint + "?routingId=" + requestDto.Permissions[0].ResourceScope.AccountIdentifier
	httpRequest, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(byt))
	if err != nil {
		return false, err
	}

	httpRequest.Header = http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{"Bearer " + a.authToken},
	}

	httpResponse, err := a.client.Do(httpRequest)
	if err != nil {
		return false, err
	}

	if httpResponse.StatusCode != 200 {
		return false, fmt.Errorf("Got unexpected status code '%d' - assume unauthorized.", httpResponse.StatusCode)
	}

	bodyByte, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		return false, err
	}

	var responseDto aclResponse
	err = json.Unmarshal(bodyByte, &responseDto)
	if err != nil {
		return false, err
	}

	return checkAclResponse(permissionChecks, responseDto)
}

func createAclRequest(principalType enum.PrincipalType, principalId string, permissionChecks []*types.PermissionCheck) (*aclRequest, error) {
	// Generate ACL req
	req := aclRequest{
		Permissions: []aclPermission{},
		Principal: aclPrincipal{
			PrincipalIdentifier: principalId,
			PrincipalType:       string(principalType),
		},
	}

	// map all permissionchecks to ACL permission checks
	for _, c := range permissionChecks {
		mappedPermission, err := mapPermission(c.Permission)
		if err != nil {
			return nil, err
		}
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

func checkAclResponse(permissionChecks []*types.PermissionCheck, responseDto aclResponse) (bool, error) {
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
			return false, fmt.Errorf("Permission '%s' is not permitted according to ACL (correlationId: '%s')",
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

	harnessIdentifiers := strings.Split(scope.SpacePath, "/")
	if len(harnessIdentifiers) > 3 {
		return nil, fmt.Errorf("Unable to convert '%s' to harness resource scope (expected {Account}/{Organization}/{Project} or a sub scope).", scope.SpacePath)
	}

	aclScope := &aclResourceScope{}
	if len(harnessIdentifiers) > 0 {
		aclScope.AccountIdentifier = harnessIdentifiers[0]
	}
	if len(harnessIdentifiers) > 1 {
		aclScope.OrgIdentifier = harnessIdentifiers[1]
	}
	if len(harnessIdentifiers) > 2 {
		aclScope.ProjectIdentifier = harnessIdentifiers[2]
	}

	return aclScope, nil
}

func mapPermission(permission enum.Permission) (string, error) {
	// harness has multiple modules - add scm prefix
	return "scm_" + string(permission), nil
}
