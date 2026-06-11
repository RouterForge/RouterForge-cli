Cypress.Commands.add('login', (email, password) => {
  cy.session([email, password], () => {
    cy.request({
      method: 'POST',
      url: '/api/users/login',
      body: { email, password },
    }).then((response) => {
      expect(response.status).to.eq(200);
      window.localStorage.setItem('token', response.body.token);
    });
  });
});

Cypress.Commands.add('createItem', (itemData) => {
  cy.request({
    method: 'POST',
    url: '/api/items',
    headers: {
      Authorization: `Bearer ${window.localStorage.getItem('token')}`,
    },
    body: itemData,
  }).then((response) => {
    expect(response.status).to.eq(201);
    return response.body;
  });
});

Cypress.Commands.add('updateItem', (itemId, updates) => {
  cy.request({
    method: 'PUT',
    url: `/api/items/${itemId}`,
    headers: {
      Authorization: `Bearer ${window.localStorage.getItem('token')}`,
    },
    body: updates,
  }).then((response) => {
    expect(response.status).to.eq(200);
    return response.body;
  });
});

Cypress.Commands.add('deleteItem', (itemId) => {
  cy.request({
    method: 'DELETE',
    url: `/api/items/${itemId}`,
    headers: {
      Authorization: `Bearer ${window.localStorage.getItem('token')}`,
    },
  }).then((response) => {
    expect(response.status).to.eq(204);
  });
});