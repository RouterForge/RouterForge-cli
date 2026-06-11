## Issue Title: [Auth] Forgot Password endpoint leaks user existence
### Severity
Medium
### Describe the bug
The "Forgot Password" form returns different HTTP responses and messages for registered vs unregistered email accounts, allowing user enumeration.
### To Reproduce
1. Navigate to `/auth/forgot-password`
2. Enter an unregistered email address (e.g., `nonexistent@test.com`)
3. Click "Send Reset Link"
4. Observe the response message
### Expected behavior
Generic success message: "If an account with that email exists, you will receive a password reset link." (HTTP 200 or 202)
### Actual behavior
Red error toast: "User not found" or "Something went wrong" (HTTP 404 or 422), confirming the email does not exist in the system.