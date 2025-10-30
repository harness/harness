/*
 * Copyright 2024 Harness Inc. All rights reserved.
 * Use of this source code is governed by the PolyForm Shield 1.0.0 license
 * that can be found in the licenses directory at the root of this repository, also available at
 * https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
 */

import { getRandomNameByType } from '../../../utils/getRandomNameByType'

function selectOptions(name: string) {
  cy.contains(name)
    .should('be.visible')
    .parent()
    .parent()
    .parent()
    .parent()
    .get('span[data-icon=Options]')
    .should('be.visible')
    .click()
}

describe('Docker registry e2e', () => {
  const projectName = getRandomNameByType('project')
  const registryName = getRandomNameByType('registry')
  const upstreamProxyNameWithDockerHubSource = getRandomNameByType('upstreamProxy')
  const upstreamProxyNameWithCustomSource = getRandomNameByType('upstreamProxy')
  const artifactName = 'alpine'
  const artifactVersion = 'latest'

  beforeEach(() => {
    cy.login()
    cy.intercept({ method: 'GET', url: 'api/v1/spaces/*/registries?*' }).as('getRegistries')
    cy.intercept({ method: 'GET', url: 'api/v1/registry/*' }).as('getRegistry')
    cy.intercept({ method: 'GET', url: 'api/v1/spaces/*/artifacts?*' }).as('getArtifacts')
    cy.intercept({ method: 'POST', url: 'api/v1/registry?*' }).as('createRegistry')
    cy.intercept({ method: 'PUT', url: 'api/v1/registry/*' }).as('updateRegistry')
    cy.intercept({ method: 'DELETE', url: 'api/v1/registry/*' }).as('deleteRegistry')
  })

  it('should create registry without any error', () => {
    cy.createProject(projectName)
    cy.navigateToRegistries(projectName)
    cy.wait('@getRegistries').its('response.statusCode').should('equal', 200)
    cy.get('div[data-testid="page-subheader"]').contains('New Artifact Registry').should('be.visible').click()
    cy.get('.bp3-dialog').within(() => {
      cy.contains('Create a New Artifact Registry').should('be.visible')

      cy.get('div[class*="ThumbnailSelect-"]').contains('Docker').should('be.visible').click()

      cy.get('input[name="identifier"]').focus().clear().type(registryName)

      cy.get('span[data-testid="description-edit"]').should('be.visible').click().wait(500)
      cy.get('textarea[name="description"]').focus().type('created from cypress automation')

      cy.get('span[data-testid="tags-edit"]').should('be.visible').click().wait(500)
      cy.get('.bp3-tag-input input').focus().clear().type('test{enter}test2{enter}test3{enter}')

      cy.get('button[type="submit"]').click()
      cy.wait('@createRegistry').its('response.statusCode').should('equal', 201)
      cy.wait('@getRegistry').its('response.statusCode').should('equal', 200)
    })
  })

  it('should show newly created regitsry in table and should update details without any error', () => {
    cy.navigateToRegistries(projectName)
    cy.wait('@getRegistries').its('response.statusCode').should('equal', 200)
    cy.contains(registryName).should('be.visible')

    cy.get('input[placeholder="Search"').focus().clear().type(registryName)
    cy.wait('@getRegistries').its('response.statusCode').should('equal', 200)
    cy.contains(registryName).should('be.visible').click()

    cy.wait('@getRegistry').its('response.statusCode').should('equal', 200)
    cy.wait('@getArtifacts').its('response.statusCode').should('equal', 200)
    cy.get('div[role="tablist"]').contains('Configuration').should('be.visible').click()

    cy.get('button[aria-label="Save"]').should('be.visible').should('be.disabled')

    cy.get('textarea[name="description"]').focus().clear()
    cy.get('span[data-testid="description-edit"]').should('be.visible').click().wait(500)
    cy.get('textarea[name="description"]').focus().type('updated description from cypress automation')

    cy.get('.bp3-tag-input input').focus().clear().type('{backspace}{backspace}test4{enter}test5{enter}')
    cy.get('button[aria-label="Save"]').should('be.visible').should('not.be.disabled').click()

    cy.wait('@updateRegistry').its('response.statusCode').should('equal', 200)
    cy.get('.bp3-toast-message').contains('Registry updated successfully')
  })

  it('should upload artifacts to newly created registry ', () => {
    cy.executeScript({
      script: 'e2e/registry/docker.sh',
      params: `--space_ref ${projectName} --registry ${registryName} --artifact ${artifactName} --version ${artifactVersion}`
    }).then(scriptId => {
      cy.log('scriptId', scriptId)
      cy.pollExecutionApi(scriptId).its('status').should('equal', 'completed')
    })
    cy.navigateToRegistry(projectName, registryName)
    cy.wait(3000)
  })

  it('should able to view artifacts inside registry', () => {
    cy.validateDockerArtifacts(projectName, registryName, artifactName, artifactVersion)
  })

  it('should able to add upstream proxy in registry', () => {
    cy.navigateToRegistry(projectName, registryName, 'configuration')
    cy.contains('Advanced (Optional)').should('be.visible').click()
    cy.get('.bp3-card').contains('Upstream Proxies')
    cy.get('button[aria-label="Configure Upstream"]').should('be.visible').click()
    cy.wait('@getRegistries').its('response.statusCode').should('equal', 200)

    // create upstream proxy with dockerhub source
    cy.get('button[aria-label="New Upstream Proxy"]').should('be.visible').click()
    cy.get('.bp3-dialog').within(() => {
      cy.contains('Create a New Upstream Proxy').should('be.visible')
      cy.get('input[type=checkbox][name=packageType][value=DOCKER]').should('be.checked').should('be.disabled')
      cy.get('input[name="identifier"]').focus().clear().type(upstreamProxyNameWithDockerHubSource)
      cy.get('input[type=radio][name="config.source"][value=Dockerhub]').should('be.checked')
      cy.get('input[type=radio][name="config.authType"][value=Anonymous]').should('be.checked')
      cy.get('button[type=submit]').should('be.visible').click()
    })
    cy.wait('@createRegistry').its('response.statusCode').should('equal', 201)
    cy.wait('@getRegistries').its('response.statusCode').should('equal', 200)

    // create upstream proxy with custom source
    cy.get('button[aria-label="New Upstream Proxy"]').should('be.visible').click()
    cy.get('.bp3-dialog').within(() => {
      cy.contains('Create a New Upstream Proxy').should('be.visible')
      cy.get('input[type=checkbox][name=packageType][value=DOCKER]').should('be.checked').should('be.disabled')
      cy.get('input[name="identifier"]').focus().clear().type(upstreamProxyNameWithCustomSource)
      cy.get('input[type=radio][name="config.source"][value=Dockerhub]').should('be.checked')
      cy.get('input[type=radio][name="config.source"][value=Custom]').should('not.be.disabled').click({ force: true })
      cy.get('input[name="config.url"]').focus().clear().type('https://registry-1.docker.io')
      cy.get('input[type=radio][name="config.authType"][value=Anonymous]').should('be.checked')
      cy.get('button[type=submit]').should('be.visible').click()
    })
    cy.wait('@createRegistry').its('response.statusCode').should('equal', 201)
    cy.wait('@getRegistries').its('response.statusCode').should('equal', 200)

    cy.get('ul[aria-label=selectable-list]')
      .should('be.visible')
      .within(() => {
        cy.contains(upstreamProxyNameWithDockerHubSource).should('be.visible').click()
        cy.contains(upstreamProxyNameWithCustomSource).should('be.visible').click()
      })
    cy.get('ul[aria-label=orderable-list]')
      .should('be.visible')
      .within(() => {
        cy.contains(upstreamProxyNameWithDockerHubSource).should('be.visible').click()
        cy.contains(upstreamProxyNameWithCustomSource).should('be.visible').click()
      })
    cy.get('button[aria-label="Save"]').should('be.visible').should('not.be.disabled').click()

    cy.wait('@updateRegistry').its('response.statusCode').should('equal', 200)
    cy.get('.bp3-toast-message').contains('Registry updated successfully')
  })

  it('should show newly created regitsry in table and able to delete without any error', () => {
    cy.navigateToRegistries(projectName)
    cy.wait('@getRegistries').its('response.statusCode').should('equal', 200)

    cy.get('input[placeholder="Search"').focus().clear().type(registryName)
    cy.wait('@getRegistries').its('response.statusCode').should('equal', 200)

    cy.contains(registryName).should('be.visible')
    selectOptions(registryName)
    cy.get('.bp3-menu-item').contains('Delete').should('be.visible').click()
    cy.get('.bp3-dialog')
      .should('be.visible')
      .within(() => {
        cy.contains('Delete Registry').should('be.visible')
        cy.get('button[aria-label=Delete]').should('be.visible').should('not.be.disabled').click()
        cy.wait('@deleteRegistry').its('response.statusCode').should('equal', 200)
      })
  })

  after(() => {
    cy.deleteProject(projectName)
  })
})
