// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package authz

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type HarnessAuthorizer struct {
	client      *http.Client
	aclEndpoint string
	authToken   string
}

func NewHarnessAuthorizer(aclEndpoint, authToken string) (*HarnessAuthorizer, error) {
	// build http client - could be injected, too
	tr := &http.Transport{
		// TODO: expose InsecureSkipVerify in config
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	return &HarnessAuthorizer{
		client:      client,
		aclEndpoint: aclEndpoint,
		authToken:   authToken,
	}, nil
}

func (this *HarnessAuthorizer) CheckForAccess(principalType PrincipalType, principalId string, resource Resource, permission Permission) error {
	return this.CheckForAccessAll(principalType, principalId, &PermissionCheck{Resource: resource, Permission: permission})
}

func (this *HarnessAuthorizer) CheckForAccessAll(principalType PrincipalType, principalId string, permissionChecks ...*PermissionCheck) error {
	if len(permissionChecks) == 0 {
		fmt.Errorf("No permission checks provided.")
	}

	requestDto, err := createAclRequest(principalType, principalId, permissionChecks)
	byt, err := json.Marshal(requestDto)
	if err != nil {
		return err
	}

	// TODO: accountId might be different!
	url := this.aclEndpoint + "?routingId=" + requestDto.Permissions[0].ResourceScope.AccountIdentifier
	httpRequest, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(byt))
	if err != nil {
		return err
	}

	httpRequest.Header = http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{"Bearer " + this.authToken},
	}

	httpResponse, err := this.client.Do(httpRequest)
	if err != nil {
		return err
	}

	if httpResponse.StatusCode != 200 {
		return fmt.Errorf("Got unexpected status code '%d' - assume unauthorized.", httpResponse.StatusCode)
	}

	bodyByte, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		return err
	}

	var responseDto aclResponse
	err = json.Unmarshal(bodyByte, &responseDto)
	if err != nil {
		return err
	}

	return checkAclResponse(permissionChecks, responseDto)
}

func createAclRequest(principalType PrincipalType, principalId string, permissionChecks []*PermissionCheck) (*aclRequest, error) {
	// Generate ACL request
	request := aclRequest{
		Permissions: []aclPermission{},
		Principal: aclPrincipal{
			PrincipalIdentifier: principalId,
			PrincipalType:       string(principalType),
		},
	}

	// map all permissionchecks to ACL permission checks
	for _, pCheck := range permissionChecks {
		mappedPermission, err := mapToHarnessResourcePermission(pCheck.Permission)
		if err != nil {
			return nil, err
		}
		mappedResourceScope, mappedResourceIdentifier, err := mapResourceIdentifierToHarnessResourceScopeAndIdentifier(pCheck.Resource.Identifier)
		if err != nil {
			return nil, err
		}

		request.Permissions = append(request.Permissions, aclPermission{
			Permission:         mappedPermission,
			ResourceScope:      *mappedResourceScope,
			ResourceType:       string(pCheck.Resource.Type),
			ResourceIdentifier: mappedResourceIdentifier,
		})
	}

	return &request, nil
}

func checkAclResponse(permissionChecks []*PermissionCheck, responseDto aclResponse) error {
	/*
	 * We are assuming two things:
	 *  - All permission checks were made for the same principal.
	 *  - Permissions inherit down the hierarchy (Account -> Organization -> Project -> Repository)
	 *
	 * Based on that, if there's any permitted result for a permission check the permission is allowed.
	 * Now we just have to ensure that all permissions are allowed
	 */

	for _, pCheck := range permissionChecks {
		permissionPermitted := false
		for _, accessControlElement := range responseDto.Data.AccessControlList {
			if string(pCheck.Permission) == accessControlElement.Permission && accessControlElement.Permitted {
				permissionPermitted = true
				break
			}
		}

		if !permissionPermitted {
			return fmt.Errorf(
				"Permission '%s' is not permitted according to ACL (correlationId: '%s').",
				pCheck.Permission,
				responseDto.CorrelationID)
		}
	}

	return nil
}

func mapResourceIdentifierToHarnessResourceScopeAndIdentifier(identifier string) (*aclResourceScope, string, error) {

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

func mapToHarnessResourcePermission(permission Permission) (string, error) {
	// harness has multiple modules - add scm prefix
	return "scm_" + string(permission), nil
}

/*
 * Classes required for harness acl.
 */
type aclRequest struct {
	Principal   aclPrincipal    `json:"principal"`
	Permissions []aclPermission `json:"permissions"`
}
type aclResponse struct {
	Status        string          `json:"status"`
	CorrelationID string          `json:"correlationId"`
	Data          aclResponseData `json:"data"`
}
type aclResponseData struct {
	Principal         aclPrincipal        `json:"principal"`
	AccessControlList []aclControlElement `json:"accessControlList"`
}
type aclControlElement struct {
	Permission    string           `json:"permission"`
	ResourceScope aclResourceScope `json:"resourceScope,omitempty"`
	ResourceType  string           `json:"resourceType"`
	Permitted     bool             `json:"permitted"`
}
type aclResourceScope struct {
	AccountIdentifier string `json:"accountIdentifier"`
	OrgIdentifier     string `json:"orgIdentifier,omitempty"`
	ProjectIdentifier string `json:"projectIdentifier,omitempty"`
}
type aclPermission struct {
	ResourceScope      aclResourceScope `json:"resourceScope,omitempty"`
	ResourceType       string           `json:"resourceType"`
	ResourceIdentifier string           `json:"resourceIdentifier"`
	Permission         string           `json:"permission"`
}
type aclPrincipal struct {
	PrincipalIdentifier string `json:"principalIdentifier"`
	PrincipalType       string `json:"principalType"`
}
