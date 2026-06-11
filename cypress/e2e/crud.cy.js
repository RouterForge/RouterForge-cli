describe('CRUD Operations', () => {
  before(() => {
    cy.login('test@example.com', 'password123');
  });

  beforeEach(() => {
    cy.visit('/items');
  });

  it('should create a new item', () => {
    cy.get('[data-testid="add-item-button"]').click();
    cy.get('input[name="title"]').type('New Item');
    cy.get('textarea[name="description"]').type('Description for new item');
    cy.get('button[type="submit"]').click();
    cy.contains('New Item').should('be.visible');
    cy.contains('Item created successfully').should('be.visible');
  });

  it('should read all items', () => {
    cy.get('[data-testid="item-list"]').should('exist');
    cy.get('[data-testid="item-card"]').should('have.length.at.least', 1);
  });

  it('should update an existing item', () => {
    cy.get('[data-testid="edit-item-button"]').first().click();
    cy.get('input[name="title"]').clear().type('Updated Title');
    cy.get('button[type="submit"]').click();
    cy.contains('Updated Title').should('be.visible');
    cy.contains('Item updated successfully').should('be.visible');
  });

  it('should delete an item', () => {
    cy.get('[data-testid="delete-item-button"]').first().click();
    cy.on('window:confirm', () => true);
    cy.contains('Item deleted successfully').should('be.visible');
    cy.get('[data-testid="item-card"]').should('have.length.lessThan', previousCount);
  });

  function previousCount() {
    let count;
    cy.get('[data-testid="item-card"]')
      .its('length')
      .then((len) => {
        count = len;
      });
    return count;
  }
});
```