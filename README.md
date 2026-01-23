# VideoStreamGo

A turnkey solution for creating your own tube site with multi-tenant video streaming capabilities.

## Features

- **Multi-tenant Architecture**: Support for multiple video streaming instances on subdomains
- **Video Management**: Upload, organize, and stream videos with categories and playlists
- **User Management**: Instance-specific user authentication and authorization
- **Admin Dashboard**: Platform-wide analytics and customer management
- **Billing Integration**: Stripe integration for subscription management
- **S3-compatible Storage**: MinIO integration for video and asset storage

## Tech Stack

- **Frontend**: Astro, React, TailwindCSS
- **Backend**: Go (Golang) with Gin framework
- **Database**: PostgreSQL
- **Caching**: Redis
- **Storage**: MinIO (S3-compatible)
- **Containerization**: Docker & Docker Compose

## Quick Start

### Prerequisites

- Docker Engine 20.10+
- Docker Compose v2.0+
- Git

### Installation

1. **Clone the Repository**

```bash
git clone http://www.github.com/seanrobmerriam/VideoStreamGo.git
cd VideoStreamGo
```

2. **Configure Environment Variables**

Copy the example environment file and customize as needed:

```bash
cp .env.example .env
```

3. **Start the Services**

Build and start all containers:

```bash
docker-compose up -d --build
```

4. **Access the Application**

- **Frontend**: http://localhost:3000
- **Platform API**: http://localhost:8080
- **Instance API**: http://localhost:8081
- **MinIO Console**: http://localhost:9001
- **MinIO API**: http://localhost:9000

### Stopping the Services

```bash
docker-compose down
```

To stop and remove all data volumes:

```bash
docker-compose down -v
```

## Services Overview

| Service | Port | Description |
|---------|------|-------------|
| frontend | 3000 | Astro-based frontend with nginx reverse proxy |
| platform-api | 8080 | Main API for platform management |
| instance-api | 8081 | API for individual tenant instances |
| postgres-master | 5432 | PostgreSQL database |
| redis | 6379 | Redis for caching and sessions |
| minio | 9000/9001 | S3-compatible object storage |

## Development

### Running Individual Services

To run a specific service without rebuilding:

```bash
docker-compose up -d <service-name>
```

View logs for a specific service:

```bash
docker-compose logs -f <service-name>
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| POSTGRES_USER | videostreamgo | PostgreSQL username |
| POSTGRES_PASSWORD | securepassword | PostgreSQL password |
| MASTER_DB_NAME | videostreamgo_master | Master database name |
| MINIO_ROOT_USER | minioadmin | MinIO root username |
| MINIO_ROOT_PASSWORD | minioadmin | MinIO root password |
| JWT_SECRET | your-jwt-secret-key | JWT signing secret |
| API_URL | http://platform-api:8080/v1 | Platform API URL |

## Project Structure

```
VideoStreamGo/
├── cmd/
│   ├── platform-api/     # Platform API entrypoint
│   └── instance-api/     # Instance API entrypoint
├── frontend/
│   ├── src/              # Astro/React source files
│   ├── public/           # Static assets
│   └── nginx.conf        # Nginx configuration
├── internal/
│   ├── config/           # Configuration loading
│   ├── database/         # Database operations
│   ├── handlers/         # HTTP handlers
│   ├── middleware/       # HTTP middleware
│   ├── models/           # Data models
│   ├── services/         # Business logic
│   └── dto/              # Data transfer objects
├── helm/                 # Kubernetes Helm charts
└── docker-compose.yml    # Docker Compose configuration
```

## API Endpoints

### Platform API (v1)

- `GET /health` - Health check
- `POST /auth/register` - Register new admin
- `POST /auth/login` - Admin login
- `GET /instances` - List all instances
- `POST /instances` - Create new instance
- `GET /billing/subscriptions` - List subscriptions

### Instance API

- `GET /health` - Health check
- `POST /users/register` - Register user
- `POST /users/login` - User login
- `GET /videos` - List videos
- `POST /videos` - Upload video
- `GET /categories` - List categories

## Testing

Run unit tests:

```bash
docker-compose exec platform-api go test ./...
docker-compose exec instance-api go test ./...
```

## License

MIT License
