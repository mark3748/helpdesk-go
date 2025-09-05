Vendored Swagger UI assets
==========================

These files are copied from swagger-ui-dist@5.11.0 to ensure reproducible Docker builds without network fetches at build time.

Files:
- swagger-ui.css
- swagger-ui-bundle.js
- swagger-ui-standalone-preset.js
- favicon-16x16.png
- favicon-32x32.png

Dockerfile.api copies this directory to /opt/helpdesk/swagger in the final image.

