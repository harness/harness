// @ts-check
///<reference path="../../src/global.d.ts" />
// ***********************************************
// This example commands.js shows you how to
// create various custom commands and overwrite
// existing commands.
//
// For more comprehensive examples of custom
// commands please read more here:
// https://on.cypress.io/custom-commands
// ***********************************************
//
//
// -- This is a parent command --
// Cypress.Commands.add('login', (email, password) => { ... })
//
//
// -- This is a child command --
// Cypress.Commands.add('drag', { prevSubject: 'element'}, (subject, options) => { ... })
//
//
// -- This is a dual command --
// Cypress.Commands.add('dismiss', { prevSubject: 'optional'}, (subject, options) => { ... })
//
//
// -- This will overwrite an existing command --
// Cypress.Commands.overwrite('visit', (originalFn, url, options) => { ... })

declare namespace Cypress {
  interface Chainable {
    clickSubmit(): void
    visitPageAssertion(className?: string): void
    getAccountIdentifier(): any
    apiRequest(
      method: string,
      endpoint: string,
      body?: any,
      queryParams?: { [key: string]: string },
      headerOptions?: HeadersInit
    ): void
    getSecret(scope: string): any
    fillName(name: string): void
    fillField(fieldName: string, value: string): void
    login(username?: string, password?: string): void
  }
}

export const activeTabClassName = '.TabNavigation--active'

Cypress.Commands.add('visitPageAssertion', (className = activeTabClassName) => {
  cy.get(className, {
    timeout: 30000
  }).should('be.visible')
  cy.wait(1000)
})
Cypress.Commands.add('clickSubmit', () => {
  cy.get('input[type="submit"]').click()
})

Cypress.Commands.add('getAccountIdentifier', () => {
  cy.location('hash').then(hash => {
    return cy.wrap(hash.split('/')[2])
  })
})

Cypress.Commands.add(
  'apiRequest',
  (
    method: string,
    endpoint: string,
    body?: unknown,
    queryParams?: { [key: string]: string },
    headerOptions?: HeadersInit
  ) => {
    cy.request(generateRequestObject(method, endpoint, body, queryParams, headerOptions))
  }
)

Cypress.Commands.add('fillField', (fieldName: string, value: string) => {
  cy.get(`[name="${fieldName}"]`).clear().type(value)
})
Cypress.Commands.add('fillName', (value: string) => {
  cy.fillField('name', value)
})
