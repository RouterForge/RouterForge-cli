## Issue Title: [API] PATCH /api/users/:id returns 500 on duplicate email constraint
### Severity
High
### Describe the bug
Updating a user's email to one that already exists in the database causes a 500 Internal Server Error and leaks the raw database constraint violation in the response body.
### To Reproduce
1. Ensure two users exist (User A with email a@a.com, User B with email b@b.com)
2. Send a PATCH request to `/api/users/{User A ID}` with payload `{"email": "b@b.com"}`
3. Observe the API response
### Expected behavior
HTTP 409 Conflict with a structured JSON error: `{"error": "Email already in use"}`
### Actual behavior
HTTP 500 Internal Server Error. Response body leaks the raw database error (e.g., `duplicate key value violates unique constraint "users_email_key"`).