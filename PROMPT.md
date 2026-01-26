# Infrastructure Integration & Security Hardening Implementation Brief

## Context
You are working on a multi-service Go application stack with two APIs (platform-api and instance-api) that currently has disconnected infrastructure components and security vulnerabilities. The codebase has configuration for Redis and MinIO but no actual implementations, shared JWT secrets creating security risks, and missing database constraints.

## Objectives
Implement proper distributed infrastructure, establish clear service boundaries, and add critical data integrity safeguards across the entire application stack.

---

## Phase 1: Redis Integration & Distributed Caching

### Task 1.1: Establish Redis Client Connections
- Install `github.com/redis/go-redis/v9` dependency
- Create `internal/cache/redis.go` with a Redis client wrapper that:
  - Initializes connection from config parameters (host, port, password, DB)
  - Implements connection pooling with sensible defaults (min 10, max 100 connections)
  - Includes exponential backoff retry logic for connection failures
  - Provides graceful shutdown on context cancellation
- Wire Redis client into both `cmd/platform-api/main.go` and `cmd/instance-api/main.go` during startup
- Add Redis health check that verifies PING/PONG before marking service as ready

### Task 1.2: Migrate Session Storage to Redis
- Locate current session management code (likely JWT or in-memory sessions)
- Implement Redis-backed session store with:
  - Key pattern: `session:{user_id}:{session_id}`
  - TTL matching JWT expiration time
  - Atomic set-if-not-exists for session creation
- Add session invalidation on logout that removes Redis keys
- Implement session refresh mechanism that extends TTL

### Task 1.3: Implement Cache-Aside Pattern
- Identify frequently accessed, read-heavy data (users, tenants, configurations)
- Create `internal/cache/cacheable.go` with generic cache wrapper:
  - Get: Check Redis → on miss, query DB → store in Redis with TTL → return
  - Set: Update DB → invalidate/update Redis
  - Delete: Remove from DB → delete from Redis
- Apply caching to at least 3 high-traffic endpoints
- Use cache keys with versioning: `v1:user:{id}`, `v1:tenant:{id}`
- Set appropriate TTLs (users: 5min, config: 15min, static data: 1hr)

### Task 1.4: Distributed Rate Limiting
- Create `internal/middleware/ratelimit.go` using Redis INCR commands
- Implement sliding window rate limiter with Redis sorted sets:
  - Key pattern: `ratelimit:{tenant_id}:{endpoint}:{window}`
  - Track requests per tenant per endpoint per time window
  - Return 429 with `Retry-After` header when limit exceeded
- Add different rate limits per tenant tier (free: 100/min, pro: 1000/min)
- Make rate limits configurable via environment variables

---

## Phase 2: MinIO Object Storage Integration

### Task 2.1: Establish MinIO Client
- Install `github.com/minio/minio-go/v7` dependency
- Create `internal/storage/minio.go` with MinIO client wrapper:
  - Initialize from config (endpoint, access key, secret key, use SSL)
  - Auto-create buckets if they don't exist on startup
  - Implement retry logic for transient S3 errors
- Create separate buckets for different content types: `uploads`, `exports`, `backups`
- Wire MinIO client into both API services during initialization

### Task 2.2: Replace Filesystem Storage
- Find all `os.Create`, `ioutil.WriteFile`, `os.Open` calls for user uploads
- Replace with MinIO `PutObject` calls:
  - Generate unique object keys with prefixes: `{tenant_id}/{user_id}/{timestamp}_{filename}`
  - Store metadata (original filename, content type, uploader) as S3 object metadata
  - Implement streaming uploads for large files (>5MB chunks)
- Update database records to store S3 object keys instead of file paths

### Task 2.3: Implement Presigned URL Pattern
- Create `internal/storage/presigned.go` with URL generation functions:
  - Upload presigned URLs (POST) with 15-minute expiration for client-side uploads
  - Download presigned URLs (GET) with 1-hour expiration for secure downloads
  - Include content-type and size restrictions in upload URLs
- Add API endpoints:
  - `POST /api/v1/storage/upload-url` → returns presigned upload URL + object key
  - `POST /api/v1/storage/download-url` → validates ownership → returns presigned download URL
- Implement ownership validation: verify requesting user has access to object

### Task 2.4: Storage Lifecycle Management
- Implement soft-delete pattern: move deleted objects to `deleted` bucket with 30-day retention
- Add background worker that permanently deletes objects after retention period
- Create storage usage tracking per tenant in Redis: `storage:{tenant_id}:bytes`
- Add storage quota enforcement before accepting uploads

---

## Phase 3: Security Hardening - JWT Isolation

### Task 3.1: Separate JWT Secrets Per Service
- Update `internal/config/config.go` to load separate secrets:
  - `PLATFORM_JWT_SECRET` for platform-api
  - `INSTANCE_JWT_SECRET` for instance-api
- Ensure secrets are cryptographically random (min 64 chars, from secure random source)
- Update token generation to embed service identifier in claims: `{"iss": "platform-api"}`

### Task 3.2: Service-Specific Token Validation
- Update JWT middleware in each service to:
  - Validate `iss` claim matches expected service
  - Reject tokens with wrong issuer immediately (before signature verification)
  - Log cross-service token attempts as security events
- Add token introspection endpoint for debugging (admin-only)

### Task 3.3: Implement Service-to-Service Authentication
- Create `internal/auth/service_token.go` for inter-service calls:
  - Generate short-lived service tokens (5-minute expiry) with specific claims: `{"iss": "platform-api", "aud": "instance-api", "scope": "create_instance"}`
  - Use asymmetric keys (RSA-2048 or ECDSA P-256) for service tokens
  - Implement token caching in Redis to avoid regenerating on every call
- Add middleware that validates service tokens on protected internal endpoints
- Document which endpoints require user tokens vs service tokens

### Task 3.4: Token Lifecycle Management
- Implement token refresh mechanism separate from primary tokens
- Add token revocation list (Redis set) for logout/compromise scenarios
- Create admin endpoint to revoke all tokens for a specific user
- Implement automatic token rotation on password change

---

## Phase 4: Database Integrity & Observability

### Task 4.1: Add Foreign Key Constraints
- Create new migration file: `XXXX_add_foreign_keys.up.sql`
- Add FK constraints for all relationship tables:
  ```sql
  ALTER TABLE instances ADD CONSTRAINT fk_instances_tenants 
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
  
  ALTER TABLE users ADD CONSTRAINT fk_users_tenants
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
  ```
- Use `ON DELETE CASCADE` for owned entities, `ON DELETE RESTRICT` for referenced entities
- Test constraints by attempting invalid operations (should fail gracefully)

### Task 4.2: Implement Cache Invalidation Strategy
- Create `internal/cache/invalidation.go` with event-driven invalidation:
  - Publish events to Redis pub/sub channel on entity changes: `cache:invalidate:{entity_type}:{id}`
  - Subscribe all service instances to invalidation channel
  - Implement pattern-based invalidation: `DEL user:*` on user update
- Add cache versioning: increment version number on schema changes
- Implement cache warming for critical data on service startup

### Task 4.3: Comprehensive Health Check Endpoints
- Create `internal/health/checker.go` with dependency checks:
  - Database: Execute `SELECT 1` with 2-second timeout
  - Redis: `PING` command with 1-second timeout
  - MinIO: List buckets with 3-second timeout
- Add endpoints:
  - `GET /health` → returns 200 if all dependencies healthy (for load balancers)
  - `GET /health/detailed` → returns JSON with per-dependency status + response times
- Include version, uptime, and memory usage in detailed health check
- Return 503 if any critical dependency fails

### Task 4.4: Observability Enhancements
- Add structured logging with correlation IDs across service calls
- Implement request tracing that logs: tenant_id, user_id, endpoint, duration, status
- Add Prometheus-compatible metrics endpoint (`/metrics`):
  - Request counters per endpoint
  - Response time histograms
  - Redis hit/miss ratios
  - Active connections per dependency
- Create dashboard-ready log format (JSON) with standardized fields

---

## Implementation Guidelines

### Code Quality Requirements
- All Redis/MinIO operations must have context timeouts (default 5s for Redis, 30s for MinIO)
- Every external call needs error handling with retries for transient failures
- Use connection pooling everywhere - no single-use connections
- Write unit tests for cache logic (mock Redis), integration tests for E2E flows
- Add graceful shutdown handlers that drain connections before exiting

### Configuration Management
- All new parameters go in `internal/config/config.go` with validation
- Provide sensible defaults; fail fast on missing critical config (secrets, endpoints)
- Support environment variables and config files; env vars take precedence
- Document every new config parameter in README

### Migration Strategy
- Create feature flags for each integration (can toggle Redis/MinIO on/off)
- Deploy with integrations disabled initially; validate connections work
- Enable one component at a time in production
- Keep fallback to in-memory/filesystem for 1 release cycle

### Testing Requirements
- Write Redis integration tests using `miniredis` for mocking
- Write MinIO tests using local MinIO server in Docker
- Test cache invalidation across multiple service instances
- Test rate limiting with concurrent requests
- Verify FK constraints prevent orphaned records

---

## Deliverables Checklist

- [ ] Redis client connected in both services with health checks
- [ ] Session storage migrated to Redis with TTL management
- [ ] Cache-aside pattern implemented for minimum 3 endpoints
- [ ] Redis-based distributed rate limiting per tenant
- [ ] MinIO client connected with auto-bucket creation
- [ ] All file storage migrated from filesystem to MinIO
- [ ] Presigned URL endpoints for uploads/downloads
- [ ] Separate JWT secrets per service with issuer validation
- [ ] Service-to-service authentication with asymmetric keys
- [ ] Database foreign key constraints added via migration
- [ ] Cache invalidation pub/sub system working
- [ ] Health check endpoints with dependency status
- [ ] Structured logging with correlation IDs
- [ ] All changes documented in code comments and README
- [ ] Integration tests passing for each component

---

## Success Criteria
- Zero in-memory caching or filesystem storage in production code paths
- All services can scale horizontally without state loss
- Token from platform-api cannot be used on instance-api
- Database referential integrity enforced at DB level
- Sub-100ms cache lookups for hot data
- Health checks accurately reflect system state
- No manual cleanup required for abandoned resources