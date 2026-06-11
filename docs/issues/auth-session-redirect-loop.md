## Issue Title: [Auth] Expired session causes infinite redirect loop
### Severity
Critical
### Describe the bug
When the user's session token expires, navigating to any authenticated page results in an infinite redirect loop between the login page and the requested page. The login form never fully renders.
### To Reproduce
1. Log in to the application
2. Wait for the session to expire (approximately 15 minutes) or manually clear the auth token
3. Click any internal navigation link
4. Observe browser URL flickering between `/login` and the target route indefinitely
### Expected behavior
User is redirected to `/login?session_expired=true`. After successful re-authentication, the user is returned to the originally requested page.
### Actual behavior
Browser enters an infinite redirect loop. The login page does not render properly, resulting in a blank white screen or browser-level error.