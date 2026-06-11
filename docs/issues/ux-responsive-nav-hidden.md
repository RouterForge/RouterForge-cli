## Issue Title: [UX] Sidebar navigation hidden on tablet viewports without toggle
### Severity
Medium
### Describe the bug
On viewport widths between 768px and 1024px (tablets and small desktops), the sidebar navigation component completely disappears. There is no hamburger menu or toggle button to access the navigation links.
### To Reproduce
1. Open the application in a browser and enable responsive design mode
2. Resize the viewport width to 900px
3. Observe the main content area; the sidebar is gone
4. Look for any navigation toggle (hamburger icon, button) – none exists
### Expected behavior
A hamburger menu icon or toggle button should be fixed at the top of the viewport to reveal the sidebar navigation on smaller screens.
### Actual behavior
The navigation sidebar is completely removed from the DOM. Users on tablet devices cannot access any navigation links, rendering the application unusable.