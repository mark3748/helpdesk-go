# Deployment Guide

This guide covers authentication configuration, security hardening, and production deployment best practices for the Helpdesk system.

## Authentication Modes

### Local Authentication (`AUTH_MODE=local`)

Local authentication uses HMAC-signed JWTs stored in HTTP-only cookies for simple deployments.

**Configuration:**
```bash
export AUTH_MODE=local
export AUTH_LOCAL_SECRET=your-secret-key-here  # Use strong random key in production
export ADMIN_PASSWORD=secure-admin-password   # Optional: creates initial admin user
```

**Cookie Details:**
- Cookie name: `hd_auth`
- Security flags: `HttpOnly`, `SameSite=Lax`, `Secure` (in production)
- Automatic admin user creation when `ADMIN_PASSWORD` is set or `ENV=dev`

**Use Cases:**
- Development environments
- Small deployments without existing identity providers
- Quick proof-of-concepts

### OIDC Authentication (`AUTH_MODE=oidc`)

OIDC authentication integrates with external identity providers using JWT tokens validated via JWKS.

**Configuration:**
```bash
export AUTH_MODE=oidc
export OIDC_ISSUER=https://auth.example.com/realms/helpdesk
export OIDC_JWKS_URL=https://auth.example.com/realms/helpdesk/protocol/openid-connect/certs
export OIDC_GROUP_CLAIM=groups  # JWT claim containing user roles
```

**Supported Identity Providers:**
- Keycloak
- Authentik
- Auth0
- Azure AD
- Any OIDC-compliant provider

**JWT Claims:**
- `groups`: Array of role names (`admin`, `manager`, `agent`)
- `sub`: User identifier
- `email`: User email address
- `name`: Display name

**Security Features:**
- JWKS key rotation with automatic refresh
- Issuer validation
- Algorithm restrictions (RS256, ES256)
- Clock skew tolerance
- Audience validation (when configured)

## Rate Limiting

The system uses Redis-backed rate limiting for consistency across multiple replicas.

### Configuration

```bash
export RATE_LIMIT_LOGIN=10        # Login attempts per minute per IP
export RATE_LIMIT_TICKETS=20      # Ticket operations per minute per user
export RATE_LIMIT_ATTACHMENTS=30  # Attachment operations per minute per user
```

### Protected Endpoints

- `POST /login`, `POST /logout` - Login rate limiting
- `POST /tickets` - Ticket creation
- `POST /tickets/:id/attachments/presign` - Attachment upload initiation
- `POST /tickets/:id/attachments` - Direct attachment upload
- `GET /tickets/:id/attachments/:attID` - Attachment downloads (optional)

### Monitoring

Rate limit rejections are tracked via Prometheus metric:
```
rate_limit_rejections_total{route="login|tickets_create|attachments_presign"}
```

### Dependencies

- **Redis required** for distributed rate limiting
- Falls back to in-memory (single replica only) if Redis unavailable
- Rate limits are per-user for authenticated endpoints, per-IP for login

## CORS Configuration

Control cross-origin requests to prevent unauthorized access.

### Configuration

```bash
export ALLOWED_ORIGINS=https://helpdesk.example.com,https://portal.example.com
```

### Security Considerations

- **Never use wildcards** (`*`) in production
- **Avoid public domains** to prevent data leaks
- **Use HTTPS origins** for secure communication
- Headers restricted to: `Authorization`, `Content-Type`, `X-Requested-With`
- `Vary: Origin` header always set for proper caching

### Effects of Misconfiguration

❌ **Dangerous:**
```bash
ALLOWED_ORIGINS=*
ALLOWED_ORIGINS=https://evil.com
```

✅ **Safe:**
```bash
ALLOWED_ORIGINS=https://helpdesk.internal.corp,https://portal.internal.corp
```

## Production Deployment Hardening

### Environment Security

1. **Use Kubernetes Secrets** for sensitive configuration:
```yaml
secrets:
  enabled: true
  data:
    DATABASE_URL: "postgres://user:pass@db:5432/helpdesk?sslmode=require"
    AUTH_LOCAL_SECRET: "32-byte-random-key"
    ADMIN_PASSWORD: "secure-password"
    MINIO_ACCESS_KEY: "access-key"
    MINIO_SECRET_KEY: "secret-key"
    SMTP_PASS: "smtp-password"
```

2. **Enable SSL/TLS everywhere:**
```bash
# Database
DATABASE_URL=postgres://user:pass@db:5432/helpdesk?sslmode=require

# MinIO/S3
MINIO_USE_SSL=true

# SMTP
SMTP_PORT=587  # Use STARTTLS
```

3. **Set appropriate timeouts:**
```bash
DB_TIMEOUT_MS=5000           # Database operations
REDIS_TIMEOUT_MS=2000        # Redis operations  
OBJECTSTORE_TIMEOUT_MS=10000 # S3/MinIO operations
```

### Network Security

1. **Ingress with TLS:**
```yaml
ingress:
  enabled: true
  className: nginx
  hosts:
    - host: helpdesk.example.com
  tls:
    - secretName: helpdesk-tls
      hosts:
        - helpdesk.example.com
```

2. **Network policies** (if supported):
```yaml
# Allow only necessary traffic between pods
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: helpdesk-network-policy
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: helpdesk
  ingress:
    - from:
        - podSelector:
            matchLabels:
              app.kubernetes.io/name: nginx-ingress
```

### Resource Limits

Set appropriate resource limits to prevent resource exhaustion:

```yaml
resources:
  limits:
    memory: "1Gi"
    cpu: "1000m"
  requests:
    memory: "512Mi"
    cpu: "500m"

workerResources:
  limits:
    memory: "512Mi"
    cpu: "500m"
  requests:
    memory: "256Mi"
    cpu: "250m"
```

### Health Checks

The system provides multiple health check endpoints:

**API Service:**
- `GET /healthz` - Basic liveness check
- `GET /readyz` - Readiness check (includes DB and object store)

**Worker Service:**
- `GET /health` - Basic liveness check (port 8081)
- `GET /ready` - Readiness check (includes DB and Redis)

**Kubernetes Configuration:**
```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: http
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /readyz
    port: http
  initialDelaySeconds: 5
  periodSeconds: 5
```

### Monitoring and Observability

1. **Prometheus Metrics:**
   - Request counters by endpoint
   - Authentication failure rates
   - Rate limit rejection counts
   - JWKS refresh metrics

2. **Structured Logging:**
   - JSON format in production (`ENV=prod`)
   - Request tracing with correlation IDs
   - Security event logging

3. **Grafana Dashboards:**
   See `docs/grafana.md` for dashboard examples

### Backup and Recovery

1. **Database Backups:**
   - Regular PostgreSQL dumps
   - Point-in-time recovery setup
   - Backup encryption

2. **Object Storage:**
   - S3 versioning enabled
   - Cross-region replication for critical attachments

3. **Configuration Backup:**
   - Kubernetes manifests in version control
   - Secrets encrypted at rest
   - Helm values backed up securely

## Troubleshooting

### Common Issues

1. **CORS Errors:**
   - Check `ALLOWED_ORIGINS` configuration
   - Verify origin matches exactly (including protocol)
   - Check browser developer tools for specific errors

2. **Authentication Failures:**
   - OIDC: Verify JWKS URL is accessible
   - Local: Check `AUTH_LOCAL_SECRET` is set
   - Check JWT expiration and clock skew

3. **Rate Limiting:**
   - Verify Redis connectivity
   - Check rate limit configuration values
   - Monitor Prometheus metrics

4. **Performance Issues:**
   - Check database connection pool settings
   - Monitor resource usage and limits
   - Review timeout configurations

### Debugging Commands

```bash
# Check health endpoints
curl -i http://api:8080/healthz
curl -i http://api:8080/readyz
curl -i http://worker:8081/health
curl -i http://worker:8081/ready

# Test authentication
curl -i -X POST http://api:8080/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"password"}'

# Check JWKS (OIDC mode)
curl -i $OIDC_JWKS_URL

# Verify CORS
curl -i -X OPTIONS http://api:8080/tickets \
  -H "Origin: https://helpdesk.example.com" \
  -H "Access-Control-Request-Method: GET"
```

## Security Checklist

- [ ] Strong secrets generated and stored securely
- [ ] TLS enabled for all external connections
- [ ] CORS configured with specific origins only
- [ ] Rate limiting enabled and monitored
- [ ] Resource limits set appropriately
- [ ] Health checks configured
- [ ] Monitoring and alerting in place
- [ ] Backup and recovery procedures tested
- [ ] Network policies implemented (if available)
- [ ] Security scanning integrated into CI/CD