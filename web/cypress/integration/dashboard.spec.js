describe('dashboard', () => {
    it('load the dashboard', () => {
        cy.visit('/')
        cy.contains('In Effect')
        cy.contains('Policy Evaluations')
        cy.contains('Failures Recorded')
    })
})
