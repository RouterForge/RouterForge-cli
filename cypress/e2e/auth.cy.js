describe('Authentication', () => {
  beforeEach(() => {
    cy.visit('/');
  });

  it('should display login and signup links', () => {
    cy.contains('Login').should('be.visible');
    cy.contains('Sign Up').should('be.visible');
  });

  it('should log in with valid credentials and redirect to dashboard', () => {
    cy.visit('/login');
    cy.get('input[name="email"]').type('test@example.com');
    cy.get('input[name="password"]').type('password123');
    cy.get('button[type="submit"]').click();
    cy.url().should('include', '/dashboard');
    cy.contains('Welcome').should('be.visible');
  });

  it('should display error for invalid login', () => {
    cy.visit('/login');
    cy.get('input[name="email"]').type('invalid@example.com');
    cy.get('input[name="password"]').type('wrongpassword');
    cy.get('button[type="submit"]').click();
    cy.contains('Invalid credentials').should('be.visible');
    cy.url().should('include', '/login');
  });

  it('should sign up a new user', () => {
    cy.visit('/signup');
    cy.get('input[name="email"]').type('newuser@example.com');
    cy.get('input[name="password"]').type('securePassword123');
    cy.get('input[name="confirmPassword"]').type('securePassword123');
    cy.get('button[type="submit"]').click();
    cy.url().should('include', '/dashboard');
    cy.contains('Welcome, newuser').should('be.visible');
  });
});