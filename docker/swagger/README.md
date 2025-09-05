Vendored Swagger UI assets (locked)
===================================

Version: swagger-ui-dist@5.11.0 (locked)

These files are copied into the repo to ensure reproducible Docker builds with no network access at build time.

Files and SHA-256 checksums:

- swagger-ui.css  93f1d44a8ee6589e7bc923c1c30e95dab867a0a8f91d2ab58f8d69258cb6aa07
- swagger-ui-bundle.js  aba894c6d9e1c56e6568fe48a7d3f6913992925826531ac87f1ebc2b03128aab
- swagger-ui-standalone-preset.js  0eca63d45dcfe5c66cfbff9613f2c733caae946028cdbd9ef89894d7f802004a
- favicon-16x16.png  af24ad604dd7b3bcda8f975ab973075f4a2f70a4087944a12f8ef8b63a3e07c2
- favicon-32x32.png  3ed612f41e050ca5e7000cad6f1cbe7e7da39f65fca99c02e99e6591056e5837

Update process:
- To upgrade, replace files from the desired swagger-ui-dist version and update the checksums above.

Dockerfile.api copies this directory to /opt/helpdesk/swagger in the final image.
