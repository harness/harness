/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { getRandomNameByType } from '../../../utils/getRandomNameByType'

describe('Docker upstream registry e2e', () => {
  const projectName = getRandomNameByType('project')
  const virtualRegistry = getRandomNameByType('registry')
  const registryNameWithDockerHubSource = getRandomNameByType('registry')
  const registryNameWithCustomSource = getRandomNameByType('registry')

  const artifactName = 'alpine'
  const artifactVersion = 'latest'
  const upstreamURL = 'duflee.com'
  const upstreamProject = 'ar-test'
  const upstreamUsername = 'admin'
  const upstreamPassword = 'Harbor12345'

  beforeEach(() => {
    cy.login()
    cy.intercept({ method: 'GET', url: 'api/v1/spaces/*/registries?*' }).as('getRegistries')
    cy.intercept({ method: 'GET', url: 'api/v1/registry/*' }).as('getRegistry')
    cy.intercept({ method: 'POST', url: 'api/v1/registry?*' }).as('createRegistry')
    cy.intercept({ method: 'PUT', url: 'api/v1/registry/*' }).as('updateRegistry')
    cy.intercept({ method: 'DELETE', url: 'api/v1/registry/*' }).as('deleteRegistry')
    cy.intercept({ method: 'POST', url: 'api/v1/secrets' }).as('createSecret')
  })

  it('should create registry with docker hub source', () => {
    cy.createProject(projectName)
    cy.navigateToRegistries(projectName)
    cy.wait('@getRegistries').its('response.statusCode').should('equal', 200)
    cy.get('div[data-testid="page-subheader"] button[class*="SplitButton--dropdown--"]').should('be.visible').click()
    cy.get('.bp3-popover')
      .should('be.visible')
      .within(() => {
        cy.contains('Upstream Proxy').should('be.visible').click()
      })
    cy.get('.bp3-dialog')
      .should('be.visible')
      .within(() => {
        cy.contains('Create a New Upstream Proxy').should('be.visible')
        cy.get('input[type=checkbox][name=packageType][value=DOCKER]').should('be.checked')
        cy.get('input[name="identifier"]').focus().clear().type(registryNameWithDockerHubSource)
        cy.get('input[type=radio][name="config.source"][value=Dockerhub]').should('be.checked')
        cy.get('input[type=radio][name="config.authType"][value=Anonymous]').should('be.checked')
        cy.get('button[type=submit]').should('be.visible').click()
      })
    cy.wait('@createRegistry').its('response.statusCode').should('equal', 201)
  })

  it('should create registry with custom source', () => {
    cy.navigateToRegistries(projectName)
    cy.wait('@getRegistries').its('response.statusCode').should('equal', 200)
    cy.get('div[data-testid="page-subheader"] button[class*="SplitButton--dropdown--"]').should('be.visible').click()
    cy.get('.bp3-popover')
      .should('be.visible')
      .within(() => {
        cy.contains('Upstream Proxy').should('be.visible').click()
      })
    cy.get('.bp3-dialog')
      .should('be.visible')
      .within(() => {
        cy.contains('Create a New Upstream Proxy').should('be.visible')
        cy.get('input[type=checkbox][name=packageType][value=DOCKER]').should('be.checked')
        cy.get('input[name="identifier"]').focus().clear().type(registryNameWithCustomSource)
        cy.get('input[type=radio][name="config.source"][value=Dockerhub]').should('be.checked')
        cy.get('input[type=radio][name="config.source"][value=Custom]').should('not.be.disabled').click({ force: true })
        cy.get('input[name="config.url"]').focus().clear().type(`https://${upstreamURL}`)
        cy.get('input[type=radio][name="config.authType"][value=Anonymous]').should('be.checked')
        cy.get('input[type=radio][name="config.authType"][value=UserPassword]').click({ force: true })
        cy.get('input[name="config.auth.userName"]').scrollIntoView().should('be.visible')
        cy.get('input[name="config.auth.userName"]').scrollIntoView().focus().clear().type(upstreamUsername)
        cy.get('button[aria-label="New Secret"]').should('be.visible').click()
      })
    cy.get('.bp3-dialog .bp3-dialog-header')
      .contains('Create a secret')
      .parent()
      .parent()
      .parent()
      .within(() => {
        cy.get('input[name="name"]').focus().clear().type(upstreamPassword)
        cy.get('textarea[name="value"]').focus().clear().type(upstreamPassword)
        cy.get('input[name="description"]').focus().clear().type('Created from cypress automation')
        cy.get('button[aria-label="Create Secret"]').should('be.visible').click()
        cy.wait('@createSecret').its('response.statusCode').should('equal', 201)
      })

    cy.get('.bp3-dialog').within(() => {
      cy.get('button[type=submit]').should('be.visible').click()
    })
    cy.wait('@createRegistry').its('response.statusCode').should('equal', 201)
  })

  it('should able to link upstream proxy to registry', () => {
    cy.createRegistry(projectName, virtualRegistry, 'DOCKER', 'VIRTUAL')
    cy.navigateToRegistries(projectName)

    cy.get('input[placeholder="Search"').focus().clear().type(virtualRegistry)
    cy.wait('@getRegistries').its('response.statusCode').should('equal', 200)
    cy.contains(virtualRegistry).should('be.visible')

    cy.navigateToRegistry(projectName, virtualRegistry, 'configuration')

    cy.contains('Advanced (Optional)').should('be.visible').click()
    cy.get('.bp3-card').contains('Upstream Proxies')
    cy.get('button[aria-label="Configure Upstream"]').should('be.visible').click()
    cy.wait('@getRegistries').its('response.statusCode').should('equal', 200)

    cy.get('ul[aria-label=selectable-list]')
      .should('be.visible')
      .within(() => {
        cy.contains(registryNameWithCustomSource).should('be.visible').click()
        cy.contains(registryNameWithDockerHubSource).should('be.visible').click()
      })
    cy.get('ul[aria-label=orderable-list]')
      .should('be.visible')
      .within(() => {
        cy.contains(registryNameWithCustomSource).should('be.visible').click()
        cy.contains(registryNameWithDockerHubSource).should('be.visible').click()
      })
    cy.get('button[aria-label="Save"]').should('be.visible').should('not.be.disabled').click()

    cy.wait('@updateRegistry').its('response.statusCode').should('equal', 200)
    cy.get('.bp3-toast-message').contains('Registry updated successfully')
  })

  it('should able to fetch the image from upstream proxy', () => {
    cy.executeScript({
      script: 'e2e/upstream/docker.sh',
      params: `--space_ref ${projectName} --registry ${virtualRegistry} --artifact ${artifactName} --version ${artifactVersion} --upstream_url ${upstreamURL} --upstream_project ${upstreamProject}`
    }).then(scriptId => {
      cy.log('scriptId', scriptId)
      cy.pollExecutionApi(scriptId).its('status').should('equal', 'completed')
    })
    // upstream proxy artifacts takes 1min to reflect on UI, so added wait
    cy.wait(20000)
    cy.navigateToRegistry(projectName, registryNameWithCustomSource)
  })

  it('should able to view artifacts inside upstream registry', () => {
    cy.validateDockerArtifacts(
      projectName,
      registryNameWithCustomSource,
      `${upstreamProject}/${artifactName}`,
      artifactVersion
    )
  })

  after(() => {
    cy.deleteRegistry(projectName, virtualRegistry)
    cy.deleteRegistry(projectName, registryNameWithDockerHubSource)
    cy.deleteRegistry(projectName, registryNameWithCustomSource)
    cy.deleteProject(projectName)
  })
})
