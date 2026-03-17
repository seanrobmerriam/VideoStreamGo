# VideoStreamGo Documentation

**Version:** 1.0  
**Last Updated:** January 2025

---

## Table of Contents

1. [Overview](#1-overview)
2. [Architecture](#2-architecture)
3. [Backend Development](#3-backend-development)
4. [Frontend Development](#4-frontend-development)
5. [Database Schema](#5-database-schema)
6. [API Reference](#6-api-reference)
7. [Deployment](#7-deployment)
8. [Configuration](#8-configuration)
9. [Testing](#9-testing)
10. [Contributing](#10-contributing)
11. [Security](#11-security)

---

## 1. Overview

VideoStreamGo is a white-label multi-tenant video streaming platform that enables customers to deploy their own branded video sharing websites. The platform follows a **subdomain-per-tenant** model where each customer receives an isolated instance with customizable branding, categories, and content management capabilities.

### Key Features

| Feature | Description |
|---------|-------------|
| **Multi-Tenancy** | Isolated databases and storage per customer instance |
| **Video Management** | Upload, organize, stream videos with categories and playlists |
| **User Management** | Instance-specific authentication and role-based access |
| **Admin Dashboard** | Platform-wide analytics and customer management |
| **Billing Integration** | Stripe integration for subscription management |
| **Custom Branding** | Per-instance theming with colors, logos, and custom CSS |
| **Video Streaming** | HLS adaptive bitrate streaming support |
| **Security** | httpOnly cookie-based authentication, tenant isolation |

### Tech Stack

| Layer | Technology |
|-------|------------|
| **Frontend** | Astro, React, TypeScript, TailwindCSS |
| **Backend** | Go (Golang) with Gin framework |
| **Database** | PostgreSQL with GORM |
| **Caching** | Redis |
| **Storage** | MinIO (S3-compatible) |
| **Containerization** | Docker & Docker Compose |
| **Orchestration** | Kubernetes (Helm) |

---

## 2. Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        VideoStreamGo                             │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   Frontend  │  │ Platform API│  │    Instance API         │  │
│  │  (Astro+Ngix│  │   (Gin)     │  │      (Gin)              │  │
│  └──────┬──────┘  └──────┬──────┘  └───────────┬─────────────┘  │
│         │                │                      │                │
│         └────────────────┼──────────────────────┘                │
│                          │                                       │
│  ┌───────────────────────┼───────────────────────────────────┐  │
│  │                   Nginx Reverse Proxy                     │  │
│  └───────────────────────┬───────────────────────────────────┘  │
│                          │                                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  PostgreSQL │  │    Redis    │  │        MinIO             │  │
│  │   (Master)  │  │   (Cache)   │  │    (S3 Storage)          │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Service Overview

| Service | Port | Description |
|---------|------|-------------|
| **frontend** | 3000 | Astro SSR frontend with nginx reverse proxy |
| **platform-api** | 8080 | Main API for platform management and customer accounts |
| **instance-api** | 8081 | API for individual tenant video instances |
| **postgres-master** | 5432 | PostgreSQL master database |
| **redis** | 6379 | Redis for caching and sessions |
| **minio** | 9000/9001 | S3-compatible object storage |

### Multi-Tenant Architecture

VideoStreamGo uses a hybrid multi-tenancy model:

- **Master Database**: Stores platform-level data (customers, subscriptions, instances)
- **Per-Instance Databases**: Each customer gets an isolated database for their video content
- **Subdomain-Based Routing**: Requests routed based on subdomain (e.g., `customer.videostreamgo.com`)

---

## 3. Backend Development

### Project Structure

```
internal/
├── cmd/
│   ├── platform-api/main.go      # Platform API entrypoint
│   └── instance-api/main.go      # Instance API entrypoint
├── config/
│   ├── config.go                 # Configuration loading
│   └── tenant.go                 # Tenant configuration
├── database/
│   ├── master.go                 # Master database connection
│   ├── instance.go               # Instance database connection
│   └── migrations/               # Database migrations
│       ├── master.go             # Master database migrations
│       └── instance.go           # Instance database migrations
├── handlers/
│   ├── platform/                 # Platform API handlers
│   │   ├── auth.go               # Admin authentication
│   │   ├── customer.go           # Customer management
│   │   ├── instance.go           # Instance management
│   │   ├── billing.go            # Billing operations
│   │   └── subscription.go       # Subscription management
│   └── instance/                 # Instance API handlers
│       ├── user.go               # User authentication
│       ├── video.go              # Video upload/management
│       ├── category.go           # Category management
│       └── comment.go            # Comment system
├── middleware/
│   ├── auth.go                   # JWT authentication
│   ├── tenant.go                 # Tenant context
│   ├── rate_limiting.go          # Rate limiting
│   ├── logging.go                # Request logging
│   └── validation.go             # Request validation
├── models/
│   ├── master/                   # Master database models
│   │   ├── customer.go
│   │   ├── instance.go
│   │   ├── subscription.go
│   │   └── admin_user.go
│   └── instance/                 # Instance database models
│       ├── user.go
│       ├── video.go
│       ├── category.go
│       └── comment.go
├── services/
│   ├── platform/                 # Platform business logic
│   │   ├── customer_service.go
│   │   ├── instance_service.go
│   │   └── subscription_service.go
│   └── instance/                 # Instance business logic
│       ├── user_service.go
│       ├── video_service.go
│       └── storage_service.go
├── dto/                          # Data transfer objects
│   ├── platform/
│   └── instance/
└── repository/                   # Data access layer
    ├── master/
    └── instance/
```

### Configuration

The application uses environment variables for configuration. See [Configuration](#8-configuration) for details.

### Adding a New Handler

1. Create the handler file in `internal/handlers/{platform|instance}/`
2. Define request/response DTOs in `internal/dto/`
3. Add routes in the route setup file
4. Add tests in the same directory

Example handler structure:

```go
package platform

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "videostreamgo/internal/types"
)

// MyHandler handles requests for my feature
type MyHandler struct {
    // dependencies
}

// NewMyHandler creates a new handler
func NewMyHandler() *MyHandler {
    return &MyHandler{}
}

// Create handles POST requests
func (h *MyHandler) Create(c *gin.Context) {
    var req CreateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_REQUEST", "Invalid request", err.Error()))
        return
    }
    
    // Business logic
    c.JSON(http.StatusCreated, types.SuccessResponse(result, "Created successfully"))
}
```

### Middleware

Available middleware in `internal/middleware/`:

| Middleware | Purpose |
|------------|---------|
| `AdminAuthMiddleware` | JWT auth for platform admins |
| `InstanceAuthMiddleware` | JWT auth for instance users |
| `RequireRole` | Role-based access control |
| `TenantMiddleware` | Extract tenant from subdomain |
| `RateLimit` | Request rate limiting |
| `RecoveryLogger` | Panic recovery with logging |
| `RequestLogger` | Request logging |
| `CORS` | Cross-origin resource sharing |
| `SecurityHeaders` | Security headers (HSTS, XSS, etc.) |

### Running the Platform API

```bash
# Development
go run cmd/platform-api/main.go

# With custom config
MASTER_DB_HOST=localhost go run cmd/platform-api/main.go
```

### Running the Instance API

```bash
# Development
go run cmd/instance-api/main.go
```

---

## 4. Frontend Development

### Project Structure

```
frontend/
├── src/
│   ├── components/
│   │   ├── admin/              # Admin dashboard components
│   │   │   ├── StatsCard.astro
│   │   │   ├── StatusBadge.astro
│   │   │   └── Table.astro
│   │   ├── common/             # Shared components
│   │   │   ├── Button.astro
│   │   │   ├── Card.astro
│   │   │   ├── Input.astro
│   │   │   ├── Modal.astro
│   │   │   └── ...
│   │   ├── layout/             # Layout components
│   │   │   ├── Header.astro
│   │   │   ├── Footer.astro
│   │   │   └── Sidebar.astro
│   │   └── video/              # Video-related components
│   │       ├── VideoCard.astro
│   │       ├── VideoGrid.astro
│   │       ├── VideoPlayer.tsx
│   │       └── VideoUploader.tsx
│   ├── layouts/
│   │   └── BaseLayout.astro    # Main layout
│   ├── pages/
│   │   ├── index.astro         # Landing page
│   │   ├── admin/              # Admin pages
│   │   │   ├── index.astro
│   │   │   ├── customers/
│   │   │   ├── instances/
│   │   │   └── analytics/
│   │   ├── auth/               # Authentication pages
│   │   │   ├── login.astro
│   │   │   └── register.astro
│   │   ├── dashboard/          # User dashboard
│   │   │   ├── instances/
│   │   │   ├── billing/
│   │   │   └── settings/
│   │   └── instance/           # Tenant instance pages
│   │       └── [subdomain]/    # Dynamic tenant routes
│   │           ├── index.astro
│   │           ├── watch/
│   │           │   └── [id].astro
│   │           └── admin/
│   ├── stores/                 # State management (Nano Stores)
│   │   ├── auth.ts
│   │   ├── tenant.ts
│   │   └── ui.ts
│   ├── lib/
│   │   ├── api.ts              # API client
│   │   ├── auth.ts             # Auth utilities
│   │   └── utils.ts            # Helper functions
│   ├── styles/
│   │   └── global.css          # Global styles
│   └── env.d.ts                # TypeScript environment declarations
├── public/                      # Static assets
├── astro.config.mjs            # Astro configuration
├── tailwind.config.mjs         # Tailwind configuration
├── tsconfig.json               # TypeScript configuration
└── package.json
```

### Development Commands

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview

# Type checking
npm run astro check
```

### Components

Components follow a consistent pattern:

```astro
---
// src/components/common/Button.astro
interface Props {
  variant?: 'primary' | 'secondary' | 'danger';
  size?: 'sm' | 'md' | 'lg';
  disabled?: boolean;
}

const { variant = 'primary', size = 'md', disabled = false } = Astro.props;
---

<button 
  class:list={[
    'btn',
    `btn-${variant}`,
    `btn-${size}`,
    { 'btn-disabled': disabled }
  ]}
  disabled={disabled}
>
  <slot />
</button>
```

### State Management

The frontend uses Nano Stores for state management:

```typescript
// stores/auth.ts
import { atom } from 'nanostores';

interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
}

export const $auth = atom<AuthState>({
  user: null,
  token: null,
  isAuthenticated: false,
});

export function setAuth(token: string, user: User) {
  $auth.set({
    token,
    user,
    isAuthenticated: true,
  });
}

export function clearAuth() {
  $auth.set({
    user: null,
    token: null,
    isAuthenticated: false,
  });
}
```

### API Client

```typescript
// lib/api.ts
const API_URL = import.meta.env.PUBLIC_API_URL || 'http://localhost:8080/v1';

interface RequestOptions extends RequestInit {
  params?: Record<string, string>;
}

export async function api<T>(
  endpoint: string,
  options: RequestOptions = {}
): Promise<T> {
  const { params, ...fetchOptions } = options;
  
  const url = new URL(`${API_URL}${endpoint}`);
  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      url.searchParams.append(key, value);
    });
  }

  const token = getAuthToken();
  if (token) {
    fetchOptions.headers = {
      ...fetchOptions.headers,
      Authorization: `Bearer ${token}`,
    };
  }

  const response = await fetch(url.toString(), fetchOptions);
  
  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.message || 'API Error');
  }

  return response.json();
}
```

### Tenant Pages

Dynamic tenant pages are located in `pages/instance/[subdomain]/`:

```astro
---
// pages/instance/[subdomain]/index.astro
import BaseLayout from '../../../layouts/BaseLayout.astro';
import VideoGrid from '../../../components/video/VideoGrid.astro';

const { subdomain } = Astro.params;

// Fetch tenant data
const tenant = await getTenant(subdomain);
---

<BaseLayout title={tenant.name}>
  <div class="tenant-header">
    <img src={tenant.logoUrl} alt={tenant.name} />
    <h1>{tenant.name}</h1>
  </div>
  
  <VideoGrid tenant={subdomain} />
</BaseLayout>
```

---

## 5. Database Schema

### Master Database Schema

The master database stores platform-level data.

#### Tables

| Table | Description |
|-------|-------------|
| `customers` | Customer accounts and billing information |
| `subscription_plans` | Available subscription tiers |
| `subscriptions` | Customer subscription records |
| `instances` | Customer video tube instances |
| `instance_config` | Per-instance configuration |
| `usage_metrics` | Resource usage tracking |
| `billing_records` | Invoice and payment history |
| `licenses` | Software licenses |
| `admin_users` | Platform administrators |
| `platform_settings` | Platform configuration |

#### Customers Table

```sql
CREATE TABLE customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    company_name VARCHAR(255) NOT NULL,
    contact_name VARCHAR(255),
    phone VARCHAR(50),
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'cancelled', 'pending')),
    stripe_customer_id VARCHAR(255),
    billing_email VARCHAR(255),
    tax_id VARCHAR(100),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

#### Instances Table

```sql
CREATE TABLE instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    subdomain VARCHAR(63) UNIQUE NOT NULL,
    custom_domains TEXT[] DEFAULT '{}',
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'provisioning', 'active', 'suspended', 'terminated')),
    plan_id UUID REFERENCES subscription_plans(id),
    database_name VARCHAR(63) NOT NULL,
    storage_bucket VARCHAR(63) NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    activated_at TIMESTAMP WITH TIME ZONE
);
```

### Instance Database Schema

Each tenant instance has its own database with the following schema.

#### Core Tables

| Table | Description |
|-------|-------------|
| `users` | Instance users |
| `videos` | Uploaded videos |
| `categories` | Video categories |
| `tags` | Video tags |
| `video_tags` | Video-tag associations |
| `comments` | Video comments |
| `ratings` | Video ratings |
| `favorites` | User favorites |
| `playlists` | User playlists |
| `playlist_videos` | Playlist-video associations |
| `video_views` | View analytics |
| `user_sessions` | User sessions |
| `branding_config` | Instance branding |
| `pages` | Custom pages |
| `settings` | Instance settings |

#### Videos Table

```sql
CREATE TABLE videos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    user_id UUID NOT NULL REFERENCES users(id),
    category_id UUID REFERENCES categories(id),
    status VARCHAR(50) DEFAULT 'processing' CHECK (status IN ('pending', 'processing', 'transcoding', 'ready', 'active', 'hidden', 'failed', 'deleted')),
    video_url VARCHAR(500) NOT NULL,
    thumbnail_url VARCHAR(500),
    hls_path VARCHAR(500),
    dash_path VARCHAR(500),
    duration DOUBLE PRECISION DEFAULT 0,
    file_size BIGINT DEFAULT 0,
    resolution VARCHAR(20),
    resolution_label VARCHAR(20),
    bitrate INTEGER DEFAULT 0,
    codec VARCHAR(50),
    audio_codec VARCHAR(50),
    frame_rate DOUBLE PRECISION DEFAULT 0,
    processing_status VARCHAR(50) DEFAULT 'pending',
    processing_progress INTEGER DEFAULT 0,
    processing_error TEXT,
    view_count BIGINT DEFAULT 0,
    like_count INTEGER DEFAULT 0,
    dislike_count INTEGER DEFAULT 0,
    comment_count INTEGER DEFAULT 0,
    is_featured BOOLEAN DEFAULT false,
    is_public BOOLEAN DEFAULT true,
    published_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);
```

### Migrations

Migrations are located in `internal/database/migrations/`:

- `master.go` - Master database migrations
- `instance.go` - Instance database migrations

Run migrations:

```bash
# Platform API auto-runs migrations on startup
docker-compose up -d platform-api
```

---

## 6. API Reference

### Platform API (v1)

Base URL: `http://localhost:8080/v1`

#### Authentication

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/auth/register` | POST | Register new admin user |
| `/auth/login` | POST | Admin login |
| `/auth/refresh` | POST | Refresh access token |
| `/auth/logout` | POST | Admin logout (invalidates cookie) |

> **Security Note:** Authentication uses httpOnly cookies. The server sets `auth_token` as an httpOnly, Secure, SameSite=Strict cookie on login. Clients must use `withCredentials: true` for cross-origin requests.

#### Customers

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/customers` | GET | List all customers |
| `/customers` | POST | Create new customer |
| `/customers/{id}` | GET | Get customer details |
| `/customers/{id}` | PUT | Update customer |
| `/customers/{id}` | DELETE | Delete customer |
| `/customers/{id}/suspend` | POST | Suspend customer |
| `/customers/{id}/activate` | POST | Activate customer |

#### Instances

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/instances` | GET | List all instances |
| `/instances` | POST | Create new instance |
| `/instances/{id}` | GET | Get instance details |
| `/instances/{id}` | PUT | Update instance |
| `/instances/{id}` | DELETE | Delete instance |
| `/instances/{id}/provision` | POST | Provision instance |
| `/instances/{id}/deprovision` | POST | Deprovision instance |
| `/instances/{id}/status` | GET | Get provisioning status |
| `/instances/{id}/domains` | POST | Add custom domain |
| `/instances/{id}/metrics` | GET | Get usage metrics |
| `/instances/{id}/suspend` | POST | Suspend instance |
| `/instances/{id}/activate` | POST | Activate instance |

#### Billing

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/billing/plans` | GET | List subscription plans |
| `/billing/subscriptions` | GET | List subscriptions |
| `/billing/subscriptions` | POST | Create subscription |
| `/billing/subscriptions/{id}` | GET | Get subscription |
| `/billing/subscriptions/{id}` | PUT | Update subscription |
| `/billing/invoices` | GET | List invoices |
| `/billing/invoices/{id}` | GET | Get invoice |

#### Admin

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/users` | GET | List admin users |
| `/admin/users` | POST | Create admin user |
| `/admin/settings` | GET | Get platform settings |
| `/admin/settings` | PUT | Update platform settings |

### Instance API

Base URL: `http://localhost:8081/v1`

#### Authentication

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/auth/register` | POST | Register user |
| `/auth/login` | POST | User login |
| `/auth/logout` | POST | User logout (invalidates cookie) |
| `/auth/me` | GET | Get current user |

> **Security Note:** Instance authentication uses httpOnly cookies for session management. Each instance has isolated user data and authentication state.

#### Users

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/users` | GET | List users (admin) |
| `/users/{id}` | GET | Get user profile |
| `/users/{id}` | PUT | Update user |
| `/users/{id}/ban` | POST | Ban user (admin) |
| `/users/{id}/unban` | POST | Unban user (admin) |

#### Videos

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/videos` | GET | List videos |
| `/videos` | POST | Upload video |
| `/videos/{id}` | GET | Get video details |
| `/videos/{id}` | PUT | Update video |
| `/videos/{id}` | DELETE | Delete video |
| `/videos/{id}/view` | POST | Record view |
| `/videos/{id}/rate` | POST | Rate video |
| `/videos/{id}/comments` | GET | Get comments |
| `/videos/upload/init` | POST | Initialize chunked upload |
| `/videos/upload/{id}/{chunk}` | PUT | Upload chunk |
| `/videos/upload/{id}/complete` | POST | Complete upload |

#### Categories

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/categories` | GET | List categories |
| `/categories` | POST | Create category |
| `/categories/{id}` | GET | Get category |
| `/categories/{id}` | PUT | Update category |
| `/categories/{id}` | DELETE | Delete category |

#### Comments

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/comments` | GET | List comments |
| `/comments` | POST | Create comment |
| `/comments/{id}` | GET | Get comment |
| `/comments/{id}` | PUT | Update comment |
| `/comments/{id}` | DELETE | Delete comment |

#### Branding

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/branding` | GET | Get instance branding |
| `/branding` | PUT | Update branding (admin) |

#### Health

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |

---

## 7. Deployment

### Docker Compose (Development)

```bash
# Start all services
docker-compose up -d --build

# View logs
docker-compose logs -f

# Stop all services
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

### Kubernetes (Helm)

```bash
# Install Helm chart
helm install videostreamgo ./helm/videostreamgo

# Upgrade
helm upgrade videostreamgo ./helm/videostreamgo

# Uninstall
helm uninstall videostreamgo
```

### Helm Values

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | 1 |
| `image.repository` | Docker image | ghcr.io/videostreamgo |
| `image.tag` | Image tag | latest |
| `resources.limits.cpu` | CPU limit | 1000m |
| `resources.limits.memory` | Memory limit | 1Gi |
| `postgres.size` | PostgreSQL storage | 50Gi |
| `redis.enabled` | Enable Redis | true |
| `minio.enabled` | Enable MinIO | true |

---

## 8. Configuration

### Environment Variables

#### Master Database

| Variable | Default | Description |
|----------|---------|-------------|
| `MASTER_DB_HOST` | localhost | Master database host |
| `MASTER_DB_PORT` | 5432 | Master database port |
| `MASTER_DB_USER` | videostreamgo | Master database user |
| `MASTER_DB_PASSWORD` | securepassword | Master database password |
| `MASTER_DB_NAME` | videostreamgo_master | Master database name |
| `MASTER_DB_SSLMODE` | require | SSL mode |

#### Instance Database

| Variable | Default | Description |
|----------|---------|-------------|
| `INSTANCE_DB_HOST` | localhost | Instance database host |
| `INSTANCE_DB_PORT` | 5432 | Instance database port |
| `INSTANCE_DB_USER` | videostreamgo | Instance database user |
| `INSTANCE_DB_PASSWORD` | securepassword | Instance database password |
| `INSTANCE_DB_PREFIX` | instance_ | Instance database name prefix |
| `INSTANCE_DB_SSLMODE` | require | SSL mode |

#### Application

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENV` | development | Environment (development/production) |
| `APP_DEBUG` | true | Debug mode |
| `APP_PORT` | 8080 | Platform API port |
| `JWT_SECRET` | **(required)** | JWT signing secret (min 32 characters) |
| `ENCRYPTION_KEY` | **(required)** | Encryption key for sensitive data (min 32 characters, AES-256) |
| `ALLOWED_ORIGINS` | **(required)** | Comma-separated CORS origins |

#### Redis

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_HOST` | localhost | Redis host |
| `REDIS_PORT` | 6379 | Redis port |
| `REDIS_PASSWORD` | - | Redis password |
| `REDIS_DATABASE` | 0 | Redis database |
| `REDIS_POOL_SIZE` | 10 | Connection pool size |

#### S3 Storage

| Variable | Default | Description |
|----------|---------|-------------|
| `S3_ENDPOINT` | localhost:9000 | S3 endpoint |
| `S3_ACCESS_KEY` | minioadmin | Access key |
| `S3_SECRET_KEY` | minioadmin | Secret key |
| `S3_BUCKET` | videostreamgo | Default bucket |
| `S3_REGION` | us-east-1 | AWS region |
| `S3_USE_SSL` | false | Use HTTPS |

#### Stripe

| Variable | Default | Description |
|----------|---------|-------------|
| `STRIPE_SECRET_KEY` | - | Stripe secret key |
| `STRIPE_PUBLISHABLE_KEY` | - | Stripe publishable key |
| `STRIPE_WEBHOOK_SECRET` | - | Stripe webhook secret |

### .env Example

```bash
# Database
MASTER_DB_HOST=postgres-master
MASTER_DB_PORT=5432
MASTER_DB_USER=videostreamgo
MASTER_DB_PASSWORD=securepassword
MASTER_DB_NAME=videostreamgo_master
MASTER_DB_SSLMODE=disable

# Redis
REDIS_HOST=redis
REDIS_PORT=6379

# S3 Storage
S3_ENDPOINT=minio:9000
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
S3_BUCKET=videostreamgo
S3_USE_SSL=false

# Application
APP_ENV=development
APP_DEBUG=true
APP_PORT=8080

# SECURITY - Required for production (min 32 characters each)
JWT_SECRET=your-super-secret-jwt-key-change-in-production
ENCRYPTION_KEY=your-256-bit-encryption-key-change-in-production

# CORS - Required: comma-separated list of allowed origins
ALLOWED_ORIGINS=https://example.com,https://app.example.com

# Stripe (optional)
STRIPE_SECRET_KEY=
STRIPE_PUBLISHABLE_KEY=
STRIPE_WEBHOOK_SECRET=
```

---

## 9. Testing

### Backend Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -coverprofile=coverage.out ./...

# Run specific package
go test ./internal/handlers/platform/...

# Run with verbose output
go test -v ./...
```

### Frontend Tests

```bash
# Install dependencies
npm install

# Run tests
npm run test

# Run tests with coverage
npm run test -- --coverage

# Run in watch mode
npm run test -- --watch
```

### Test Structure

```
internal/
├── handlers/
│   └── platform/
│       ├── auth_test.go
│       ├── customer_test.go
│       └── instance_test.go
└── middleware/
    ├── auth_test.go
    └── rate_limiting_test.go
```

### Writing Tests

```go
func TestCustomerHandler_List(t *testing.T) {
    // Setup
    db := setupTestDB()
    handler := NewCustomerHandler(customerRepo)
    
    // Create test data
    customer := &master.Customer{
        Email:       "test@example.com",
        CompanyName: "Test Company",
    }
    db.Create(customer)
    
    // Execute
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Request = httptest.NewRequest("GET", "/customers", nil)
    
    handler.List(c)
    
    // Assert
    assert.Equal(t, http.StatusOK, w.Code)
}
```

---

## 10. Contributing

### Development Workflow

1. **Fork** the repository
2. **Clone** your fork
3. **Create** a feature branch
4. **Make** your changes
5. **Run** tests and linting
6. **Submit** a pull request

### Code Style

#### Go

- Use `gofmt` for formatting
- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use meaningful variable names
- Add comments for exported functions

#### Frontend

- Follow ESLint configuration
- Use Prettier for formatting
- Follow component patterns
- Add TypeScript types

### Commit Messages

```
type(scope): subject

body

footer
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

Example:
```
feat(instance): add custom domain support

- Add custom domain validation
- Implement SSL certificate provisioning
- Update tenant middleware for domain routing

Closes #123
```

### Pull Request Checklist

- [ ] Tests pass
- [ ] Code is properly formatted
- [ ] Documentation is updated
- [ ] No linting errors
- [ ] Changes are minimal and focused

---

## 11. Security

### Security Improvements

VideoStreamGo has implemented several security enhancements to protect user data and ensure proper tenant isolation.

#### Token Storage (httpOnly Cookies)

**Previous:** Tokens were stored in localStorage, vulnerable to XSS attacks.

**Current:** Tokens are now stored in httpOnly cookies with the following attributes:

```
Set-Cookie: auth_token=<token>; HttpOnly; Secure; SameSite=Strict; Path=/; Max-Age=86400
```

**Benefits:**
- Cannot be accessed by JavaScript (prevents XSS token theft)
- Automatically sent with requests
- Protected by SameSite=Strict (CSRF prevention)
- Requires Secure flag in production (HTTPS)

**Frontend Requirements:**
- Use `withCredentials: true` for cross-origin API requests
- Server handles cookie creation and invalidation on logout

#### Multi-Tenant Isolation

The platform implements robust tenant isolation through:

1. **Database Isolation**: Each tenant has a separate database (`instance_<uuid>`)
2. **Storage Isolation**: Each tenant has a dedicated S3 bucket (`tenant-<uuid>`)
3. **Subdomain-Based Routing**: Tenant context extracted from request subdomain
4. **Platform Domain Protection**: Platform domains (admin, api, www) are explicitly blocked from tenant access

#### Authentication & Authorization

- **JWT Validation**: All tokens are validated with HS256 algorithm
- **Role-Based Access Control (RBAC)**: Admin users have defined roles (super_admin, admin, viewer)
- **Admin Status Verification**: Inactive admins cannot authenticate
- **Instance-Level Authorization**: Users can only access their instance's resources

### Required Environment Variables

The following environment variables are **required** for production deployments:

| Variable | Minimum Length | Purpose |
|----------|---------------|---------|
| `JWT_SECRET` | 32 characters | JWT token signing (HS256) |
| `ENCRYPTION_KEY` | 32 characters | AES-256 encryption for sensitive data |
| `ALLOWED_ORIGINS` | - | Comma-separated list of permitted CORS origins |

**Important:**
- `JWT_SECRET` and `ENCRYPTION_KEY` must be at least 32 characters
- Use cryptographically secure random values in production
- `ALLOWED_ORIGINS` should explicitly list all allowed origins (no wildcards in production)

### CORS Configuration

The `ALLOWED_ORIGINS` environment variable controls Cross-Origin Resource Sharing:

```bash
# Development (multiple local ports)
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:4321

# Production (explicit domains)
ALLOWED_ORIGINS=https://example.com,https://app.example.com,https://admin.example.com
```

**Security Note:** Never use wildcard origins (`*`) in production. Always explicitly list allowed origins.

### Security Headers

The platform implements the following security headers:

| Header | Value | Purpose |
|--------|-------|---------|
| `X-Content-Type-Options` | nosniff | Prevent MIME type sniffing |
| `X-Frame-Options` | DENY | Prevent clickjacking |
| `X-XSS-Protection` | 1; mode=block | XSS filter (legacy browsers) |
| `Strict-Transport-Security` | max-age=31536000 | Enforce HTTPS |

### Rate Limiting

API endpoints are protected by rate limiting to prevent abuse:

- Default: 100 requests per minute per IP
- Configurable via `RATE_LIMIT_REQUESTS` and `RATE_LIMIT_WINDOW`

---

## License

MIT License - see LICENSE file for details.
