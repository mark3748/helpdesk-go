# Task Completion Checklist

## After Making Code Changes

### Testing
1. **Run Tests**: `TEST_BYPASS_AUTH=true go test -cover ./...`
2. **Check Coverage**: Ensure coverage stays ≥70% across packages
3. **Test Specific Changes**: Run targeted tests for modified packages

### Code Quality
1. **Module Hygiene**: Run `go mod tidy`
2. **Verify Clean State**: Check `git diff --exit-code go.mod go.sum`

### Frontend (if applicable)
1. **Lint**: `npm run lint` in relevant web/ directory
2. **Build**: `npm run build` to verify no build errors
3. **API Types**: Regenerate if OpenAPI spec changed (`npm run gen:api`)

### Docker (if applicable)
1. **Build Images**: Verify Docker builds succeed
2. **Test Compose**: `docker compose up` for integration testing

### Deployment (if applicable)
1. **Helm Lint**: `helm lint helm/helpdesk`
2. **Template Verification**: `helm template test helm/helpdesk`

## Before Committing
- Ensure all tests pass
- Verify no secrets or sensitive data in code
- Check that migrations are backward compatible
- Validate OpenAPI spec if API changes made

## Environment Variables to Test With
- `TEST_BYPASS_AUTH=true` for running tests
- `ENV=dev` for development mode
- `DATABASE_URL` pointing to test database
- `AUTH_MODE=local` for local development