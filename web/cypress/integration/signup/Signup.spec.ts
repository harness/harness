import {
  CreateUserResponse,
  GetUserResponse,
  membershipGetCall,
  membershipQueryGetCall,
  signupPostCall,
  userGetCall
} from './constants'

describe('Signup', () => {
  beforeEach(() => {
    cy.on('uncaught:exception', () => false)
    cy.intercept('POST', signupPostCall, CreateUserResponse).as('signupPostCall')
    cy.intercept('GET', userGetCall, GetUserResponse).as('userGetCall')
    cy.intercept('GET', membershipGetCall, []).as('membershipGetCall')
    cy.intercept('GET', membershipQueryGetCall, []).as('membershipQueryGetCall')
  })
  it('should signup a new user', () => {
    cy.visit('/register')
    cy.contains('div p', 'Sign Up').should('be.visible')
    cy.contains('span', 'User ID').should('be.visible')
    cy.contains('span', 'Email').should('be.visible')
    cy.contains('span', 'Password').should('be.visible')
    cy.contains('span', 'Confirm Password').should('be.visible')

    // Fill fields
    cy.get('input[name="username"]').type('testuser')
    cy.get('input[name="email"]').type('test@harness.io')
    cy.get('input[name="password"]').type('password')
    cy.get('input[name="confirmPassword"]').type('password')

    //click signup
    cy.contains('button span', 'Sign Up').click()
    cy.wait('@signupPostCall')
    cy.wait('@userGetCall')
    cy.get('@signupPostCall').its('request.body').should('deep.equal', {
      display_name: 'testuser',
      email: 'test@harness.io',
      uid: 'testuser',
      password: 'password'
    })
    cy.wait('@membershipGetCall')
    cy.wait('@membershipQueryGetCall')
  })
})
