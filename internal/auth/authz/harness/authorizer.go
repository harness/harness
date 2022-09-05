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

func (a *Authorizer) Check(principalType enum.PrincipalType, principalId string, resource types.Resource, permission enum.Permission) (bool, error) {
	return a.CheckAll(principalType, principalId, &types.PermissionCheck{Resource: resource, Permission: permission})
}

func (a *Authorizer) CheckAll(principalType enum.PrincipalType, principalId string, permissionChecks ...*types.PermissionCheck) (bool, error) {
	if len(permissionChecks) == 0 {
		return false, fmt.Errorf("No permission checks provided.")
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
		mappedResourceScope, mappedResourceIdentifier, err := mapResourceIdentifier(c.Resource.Identifier)
		if err != nil {
			return nil, err
		}

		req.Permissions = append(req.Permissions, aclPermission{
			Permission:         mappedPermission,
			ResourceScope:      *mappedResourceScope,
			ResourceType:       string(c.Resource.Type),
			ResourceIdentifier: mappedResourceIdentifier,
		})
	}

	return &req, nil
}

func checkAclResponse(permissionChecks []*types.PermissionCheck, responseDto aclResponse) (bool, error) {
	/*
	 * We are assuming two things:
	 *  - All permission checks were made for the same principal.
	 *  - Permissions inherit down the hierarchy (Account -> Organization -> Project -> Repository)
	 *
	 * Based on that, if there's any permitted result for a permission check the permission is allowed.
	 * Now we just have to ensure that all permissions are allowed
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
			return false, fmt.Errorf(
				"Permission '%s' is not permitted according to ACL (correlationId: '%s').",
				check.Permission,
				responseDto.CorrelationID)
		}
	}

	return true, nil
}

func mapResourceIdentifier(identifier string) (*aclResourceScope, string, error) {

	/*
	 * For now we assume only repository access to be managed by ACL.
	 * Thus, the identifier is expected to be restricted to:
	 *      {Account}/{Organization}/{Project}/{Repository}
	 * which will lead to the following output:
	 *   - AclScope: {Account} {Organization} {Project}
	 *   - AclId: {Respository}
	 *
	 * TODO: Extend once account / org / project level SCM resources are available (like accesstoken, ...)
	 */

	harnessIdentifiers := strings.Split(identifier, "/")
	if len(harnessIdentifiers) != 4 {
		return nil, "", fmt.Errorf("Unable to convert '%s' to harness resource scope (expected {Account}/{Organization}/{Project}/{Repository}).", identifier)
	}

	scope := aclResourceScope{
		AccountIdentifier: harnessIdentifiers[0],
		OrgIdentifier:     harnessIdentifiers[1],
		ProjectIdentifier: harnessIdentifiers[2],
	}

	return &scope, harnessIdentifiers[3], nil
}

func mapPermission(permission enum.Permission) (string, error) {
	// harness has multiple modules - add scm prefix
	return "scm_" + string(permission), nil
}
