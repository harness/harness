describe('policies', () => {
    it('load the table', () => {
        cy.visit('/')
        cy.contains('Policies').click()
        cy.get('[class="TableV2--row TableV2--card TableV2--clickable"]').should('have.length', 12)
    })
})
