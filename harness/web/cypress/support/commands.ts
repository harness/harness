/*
 * Copyright 2024 Harness Inc. All rights reserved.
 * Use of this source code is governed by the PolyForm Shield 1.0.0 license
 * that can be found in the licenses directory at the root of this repository, also available at
 * https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
 */

import path from 'path'
import { generateRequestObject } from '../utils/generateRequestObject'
import { PACKAGE_TYPE, REGISTRY_TYPE } from '../utils/types'
import { getRandomCreateRegistryBody } from '../utils/getRequestBodies'

const POLLING_INTERVAL = 1000 // 1 second
const TIMEOUT = 3 * 60000 // 1 minute
declare global {
  namespace Cypress {
    interface Chainable {
      apiRequest(
        method: string,
        endpoint: string,
        body?: any,
        queryParams?: { [key: string]: string | boolean },
        headerOptions?: HeadersInit,
        failOnStatusCode?: boolean,
        timeout?: number,
        origin?: string
      ): Chainable<Response<any>>
      login(): Chainable<void>
      createProject(name: string): Chainable<void>
      deleteProject(name: string): Chainable<void>
      createRegistry(space: string, name: string, packageType: PACKAGE_TYPE, type: REGISTRY_TYPE): Chainable<void>
      deleteRegistry(space: string, name: string): Chainable<void>
      navigateToRegistries(space: string): Chainable<void>
      navigateToRegistry(space: string, registry: string, tab?: string): Chainable<void>
      navigateToArtifactList(space: string, registry: string): Chainable<void>
      navigateToArtifactDetails(space: string, registry: string, artifact: string): Chainable<void>
      navigateToVersionDetails(space: string, registry: string, artifact: string, version: string): Chainable<void>
      validateDockerArtifacts(space: string, registry: string, artifact: string, version: string): Chainable<void>
      validateDockerArtifactDetails(space: string, registry: string, artifact: string, version: string): Chainable<void>
      validateDockerArtifactVersionDetails(
        space: string,
        registry: string,
        artifact: string,
        version: string
      ): Chainable<void>
      validateHelmArtifacts(space: string, registry: string, artifact: string, version: string): Chainable<void>
      validateHelmArtifactDetails(space: string, registry: string, artifact: string, version: string): Chainable<void>
      validateHelmArtifactVersionDetails(
        space: string,
        registry: string,
        artifact: string,
        version: string
      ): Chainable<void>
      executeScript(queryParams: Record<string, string>): Chainable<string>
      pollExecutionApi(scriptId: string): Chainable<any>
      pollApi(
        endpoint: string,
        validate: (res: any) => boolean,
        interval?: number,
        timeout?: number
      ): Chainable<Response<any>>
    }
  }
}

export const activeTabClassName = '.TabNavigation--active'

Cypress.Commands.add(
  'apiRequest',
  (
    method: string,
    endpoint: string,
    body?: unknown,
    queryParams?: { [key: string]: string },
    headerOptions?: HeadersInit,
    failOnStatusCode?: boolean,
    timeout?: number,
    origin?: string
  ) => {
    cy.request({
      ...generateRequestObject(method, endpoint, body, queryParams, headerOptions, origin),
      failOnStatusCode,
      timeout
    }).as(endpoint)
  }
)

Cypress.Commands.add('executeScript', (queryParams: Record<string, string>) => {
  cy.apiRequest('GET', 'execute-script', null, queryParams, null, true, null, 'http://localhost:3001').then(res => {
    cy.wrap(res.body.scriptId)
  })
})

Cypress.Commands.add('pollExecutionApi', (scriptId: string) => {
  cy.pollApi(`http://localhost:3001/script-status/${scriptId}`, res => res.body.status === 'completed', 3000).then(
    res => {
      cy.wrap({ status: res.body.status })
    }
  )
})

Cypress.Commands.add(
  'pollApi',
  (
    endpoint: string,
    validate: (res: any) => boolean,
    interval: number = POLLING_INTERVAL,
    timeout: number = TIMEOUT
  ) => {
    const startTime = new Date().getTime()

    function makeRequest() {
      return cy.request(endpoint).then(response => {
        if (validate(response)) {
          // If the validation is successful, return the response
          return response
        } else if (new Date().getTime() - startTime > timeout) {
          // If the timeout is exceeded, throw an error
          throw new Error('Polling timed out')
        } else {
          // If validation fails, wait for the interval and make the request again
          return cy.wait(interval).then(makeRequest)
        }
      })
    }

    return makeRequest()
  }
)

Cypress.Commands.add('login', () => {
  cy.visit('/')
  cy.intercept({
    method: 'POST',
    url: 'api/v1/login?**'
  }).as('login')

  // TODO: move this to config
  cy.get('input[name="username"]').focus().clear().type('admin')
  cy.get('input[name="password"]').focus().clear().type('changeit')
  cy.get('button[type="submit"]').click()

  cy.wait('@login', { timeout: 60000 }).its('response.statusCode').should('equal', 200)
  cy.visit('/')
})

Cypress.Commands.add('createProject', (name: string) => {
  cy.intercept({
    method: 'POST',
    url: `api/v1/spaces`
  }).as('createSpace')
  cy.intercept({
    method: 'GET',
    url: `api/v1/spaces/${name}`
  }).as('getSpaceDetails')
  cy.get('div[role="button"][class*="SpaceSelector-"').should('be.visible').click()
  cy.contains('New Project').should('be.visible').click()
  cy.get('input[name="name"]').focus().clear().type(name)
  cy.get('input[name="description"]').focus().clear().type('Created from cypress')
  cy.get('button[type="submit"]').click()
  cy.wait('@createSpace').its('response.statusCode').should('equal', 201)
  cy.wait('@getSpaceDetails').its('response.statusCode').should('equal', 200)
})

Cypress.Commands.add('deleteProject', (name: string) => {
  cy.apiRequest('DELETE', 'api/v1/spaces', name, {}, undefined, false)
})

Cypress.Commands.add('deleteRegistry', (space: string, name: string) => {
  const registryRef = encodeURIComponent(`${space}/${name}`)
  cy.apiRequest('DELETE', `api/v1/registry/${registryRef}`)
})

Cypress.Commands.add(
  'createRegistry',
  (space: string, name: string, packageType: PACKAGE_TYPE, type: REGISTRY_TYPE) => {
    const requestBody = getRandomCreateRegistryBody(name, packageType, type)
    cy.apiRequest('POST', `api/v1/registry?parent_ref=${space}`, requestBody).its('status').should('equal', 201)
  }
)

Cypress.Commands.add('navigateToRegistries', (space: string) => {
  cy.visit(`/spaces/${space}/registries`)
})

Cypress.Commands.add('navigateToRegistry', (space: string, registry: string, tab = 'packages') => {
  cy.visit(`/spaces/${space}/registries/${registry}?tab=${tab}`)
})

Cypress.Commands.add('navigateToArtifactList', (space: string, registry: string) => {
  cy.visit(`/spaces/${space}/registries/${registry}?tab=packages`)
})

Cypress.Commands.add('navigateToArtifactDetails', (space: string, registry: string, artifact: string) => {
  cy.visit(`/spaces/${space}/registries/${registry}/artifacts/${artifact}`)
})

Cypress.Commands.add(
  'navigateToVersionDetails',
  (space: string, registry: string, artifact: string, version: string) => {
    cy.visit(`/spaces/${space}/registries/${registry}/artifacts/${artifact}/versions/${version}`)
  }
)

Cypress.Commands.add(
  'validateDockerArtifactVersionDetails',
  (space: string, registry: string, artifact: string, version: string) => {
    cy.intercept({ method: 'GET', url: 'api/v1/registry/*/artifact/*/version/*/summary' }).as('getVersionSummary')
    cy.intercept({ method: 'GET', url: '/api/v1/registry/*/artifact/*/version/*/docker/details?*' }).as(
      'getDockerVersionDetails'
    )
    cy.intercept({ method: 'GET', url: '/api/v1/registry/*/artifact/*/version/*/docker/layers?*' }).as(
      'getDockerVersionLayers'
    )
    cy.intercept({ method: 'GET', url: '/api/v1/registry/*/artifact/*/version/*/docker/manifest?*' }).as(
      'getDockerVersionManifest'
    )

    cy.navigateToVersionDetails(space, registry, artifact, version)
    cy.wait('@getVersionSummary').its('response.statusCode').should('equal', 200)
    cy.wait('@getDockerVersionDetails').its('response.statusCode').should('equal', 200)
    cy.wait('@getDockerVersionLayers').its('response.statusCode').should('equal', 200)

    cy.get('button[aria-label=Manifest]').should('be.visible').click()
    cy.wait('@getDockerVersionManifest').its('response.statusCode').should('equal', 200)
  }
)

Cypress.Commands.add(
  'validateDockerArtifactDetails',
  (space: string, registry: string, artifact: string, version: string) => {
    cy.intercept({ method: 'GET', url: 'api/v1/registry/*/artifact/*/summary' }).as('getArtifactSummary')
    cy.intercept({ method: 'GET', url: 'api/v1/registry/*/artifact/*/versions?*' }).as('getArtifactVersions')
    cy.intercept({ method: 'GET', url: 'api/v1/registry/*/artifact/*/version/*/docker/manifests' }).as(
      'getDockerVersionManifests'
    )
    cy.intercept({ method: 'GET', url: 'api/v1/registry/*/artifact/*/version/*/summary' }).as('getVersionSummary')
    cy.intercept({ method: 'GET', url: '/api/v1/registry/*/artifact/*/version/*/docker/details?*' }).as(
      'getDockerVersionDetails'
    )
    cy.intercept({ method: 'GET', url: '/api/v1/registry/*/artifact/*/version/*/docker/layers?*' }).as(
      'getDockerVersionLayers'
    )

    cy.navigateToArtifactDetails(space, registry, artifact)

    cy.wait('@getArtifactSummary').its('response.statusCode').should('equal', 200)
    cy.wait('@getArtifactVersions').its('response.statusCode').should('equal', 200)
    cy.get('div[class*="TableV2--cells--"] div[class*="TableV2--cell--"]').eq(1).contains(version).should('be.visible')
    cy.get('input[placeholder="Search"').focus().clear().type(version)
    cy.wait('@getArtifactVersions').its('response.statusCode').should('equal', 200)
    cy.get('div[class*="TableV2--cells--"] div[class*="TableV2--cell--"]')
      .eq(1)
      .contains(version)
      .should('be.visible')
      .click()

    // digest list
    cy.wait('@getDockerVersionManifests').its('response.statusCode').should('equal', 200)
    cy.get('div[class*="DigestListTable-"] div[class*="TableV2--cells--"] div[class*="TableV2--cell--"] a').click()

    // version details page
    cy.wait('@getVersionSummary').its('response.statusCode').should('equal', 200)
    cy.wait('@getDockerVersionDetails').its('response.statusCode').should('equal', 200)
    cy.wait('@getDockerVersionLayers').its('response.statusCode').should('equal', 200)

    cy.validateDockerArtifactVersionDetails(space, registry, artifact, version)
  }
)

Cypress.Commands.add(
  'validateDockerArtifacts',
  (space: string, registry: string, artifact: string, version: string) => {
    cy.intercept({ method: 'GET', url: 'api/v1/registry/*' }).as('getRegistry')
    cy.intercept({ method: 'GET', url: 'api/v1/spaces/*/artifacts?*' }).as('getArtifacts')
    cy.intercept({ method: 'GET', url: 'api/v1/registry/*/artifact/*/summary' }).as('getArtifactSummary')
    cy.intercept({ method: 'GET', url: 'api/v1/registry/*/artifact/*/versions?*' }).as('getArtifactVersions')

    cy.navigateToArtifactList(space, registry)
    cy.wait('@getRegistry').its('response.statusCode').should('equal', 200)
    cy.wait('@getArtifacts').its('response.statusCode').should('equal', 200)
    cy.contains(artifact).should('be.visible').should('be.visible')

    cy.get('input[placeholder="Search"').focus().clear().type(artifact)
    cy.wait('@getArtifacts').its('response.statusCode').should('equal', 200)
    cy.contains(artifact).should('be.visible').should('be.visible').click()

    // artifact details page
    cy.wait('@getArtifactSummary').its('response.statusCode').should('equal', 200)
    cy.wait('@getArtifactVersions').its('response.statusCode').should('equal', 200)
    cy.validateDockerArtifactDetails(space, registry, artifact, version)
  }
)

Cypress.Commands.add(
  'validateHelmArtifactVersionDetails',
  (space: string, registry: string, artifact: string, version: string) => {
    cy.intercept({ method: 'GET', url: 'api/v1/registry/*/artifact/*/version/*/summary' }).as('getVersionSummary')
    cy.intercept({ method: 'GET', url: '/api/v1/registry/*/artifact/*/version/*/helm/details' }).as(
      'getHelmVersionDetails'
    )
    cy.intercept({ method: 'GET', url: '/api/v1/registry/*/artifact/*/version/*/helm/manifest' }).as(
      'getHelmVersionManifest'
    )

    cy.navigateToVersionDetails(space, registry, artifact, version)
    cy.wait('@getVersionSummary').its('response.statusCode').should('equal', 200)
    cy.wait('@getHelmVersionDetails').its('response.statusCode').should('equal', 200)
    cy.wait('@getHelmVersionManifest').its('response.statusCode').should('equal', 200)
  }
)

Cypress.Commands.add(
  'validateHelmArtifactDetails',
  (space: string, registry: string, artifact: string, version: string) => {
    cy.intercept({ method: 'GET', url: 'api/v1/registry/*/artifact/*/summary' }).as('getArtifactSummary')
    cy.intercept({ method: 'GET', url: 'api/v1/registry/*/artifact/*/versions?*' }).as('getArtifactVersions')
    cy.intercept({ method: 'GET', url: 'api/v1/registry/*/artifact/*/version/*/summary' }).as('getVersionSummary')
    cy.intercept({ method: 'GET', url: '/api/v1/registry/*/artifact/*/version/*/helm/details' }).as(
      'getHelmVersionDetails'
    )
    cy.intercept({ method: 'GET', url: '/api/v1/registry/*/artifact/*/version/*/helm/manifest' }).as(
      'getHelmVersionManifest'
    )

    cy.navigateToArtifactDetails(space, registry, artifact)
    cy.wait('@getArtifactSummary').its('response.statusCode').should('equal', 200)
    cy.wait('@getArtifactVersions').its('response.statusCode').should('equal', 200)
    cy.get('div[class*="TableV2--cells--"] div[class*="TableV2--cell--"]').eq(0).contains(version).should('be.visible')
    cy.get('input[placeholder="Search"').focus().clear().type(version)
    cy.wait('@getArtifactVersions').its('response.statusCode').should('equal', 200)
    cy.get('div[class*="TableV2--cells--"] div[class*="TableV2--cell--"]')
      .eq(0)
      .contains(version)
      .should('be.visible')
      .click()

    // version details page
    cy.wait('@getVersionSummary').its('response.statusCode').should('equal', 200)
    cy.wait('@getHelmVersionDetails').its('response.statusCode').should('equal', 200)
    cy.wait('@getHelmVersionManifest').its('response.statusCode').should('equal', 200)
  }
)

Cypress.Commands.add('validateHelmArtifacts', (space: string, registry: string, artifact: string, version: string) => {
  cy.intercept({ method: 'GET', url: 'api/v1/registry/*' }).as('getRegistry')
  cy.intercept({ method: 'GET', url: 'api/v1/spaces/*/artifacts?*' }).as('getArtifacts')
  cy.intercept({ method: 'GET', url: 'api/v1/registry/*/artifact/*/summary' }).as('getArtifactSummary')
  cy.intercept({ method: 'GET', url: 'api/v1/registry/*/artifact/*/versions?*' }).as('getArtifactVersions')

  cy.navigateToArtifactList(space, registry)
  cy.wait('@getRegistry').its('response.statusCode').should('equal', 200)
  cy.wait('@getArtifacts').its('response.statusCode').should('equal', 200)
  cy.contains(artifact).should('be.visible').should('be.visible')

  cy.get('input[placeholder="Search"').focus().clear().type(artifact)
  cy.wait('@getArtifacts').its('response.statusCode').should('equal', 200)
  cy.contains(artifact).should('be.visible').should('be.visible').click()

  // artifact details page
  cy.wait('@getArtifactSummary').its('response.statusCode').should('equal', 200)
  cy.wait('@getArtifactVersions').its('response.statusCode').should('equal', 200)
})
