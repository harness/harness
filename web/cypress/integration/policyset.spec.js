describe('policy sets', () => {
    it('load the table', () => {
        cy.visit('/')
        cy.contains('Policy Set').click()
        cy.contains('A harness policy set allows you to group policies and configure where they will be enforced.')
    })
})
