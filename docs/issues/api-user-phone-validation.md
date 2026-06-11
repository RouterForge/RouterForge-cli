## Issue Title: [API] POST /api/users accepts invalid phone number input
### Severity
Low
### Describe the bug
The user creation and update endpoints do not validate the phone number field. Non-numeric strings and invalid formats are accepted and stored in the database without any sanitization.
### To Reproduce
1. Send a POST request to `/api/users` with payload `{"phone": "not_a_phone_number"}`
2. Retrieve the created user
3. Observe the phone number field in the response
### Expected behavior
API returns HTTP 422 Unprocessable Entity with a validation error body for the phone field. Field should only accept valid phone number formats (e.g., E.164).
### Actual behavior
API returns HTTP 201 Created. The garbage value `"not_a_phone_number"` is stored in the phone column of the users table.