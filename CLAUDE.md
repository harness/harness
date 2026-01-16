# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This is the Drone CI/CD server repository, a lightweight, Docker-native continuous integration platform written in Go. The codebase follows a modular architecture with clean separation between API handlers, business logic services, and data storage layers.

## Build System

### Primary Build Tool: Go with Task Runner

**Environment Setup:**
```bash
export GO111MODULE=on
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0
```

**Task Runner Commands (Taskfile.yml):**
```bash
# Install the server binary
task install

# Build for Linux AMD64
task build

# Run all tests
task test

# Test with MySQL backend
task test-mysql

# Test with PostgreSQL backend  
task test-postgres

# Build Docker images
task docker

# Clean build artifacts
task cleanup
```

**Manual Go Commands:**
```bash
# Build the server
go build -o drone-server github.com/drone/drone/cmd/drone-server

# Run tests
go test ./...
go test -race ./...

# Build with specific tags
go build -tags "oss nolimit" github.com/drone/drone/cmd/drone-server

# Install dependencies
go mod download
go mod tidy
```

### Build Scripts
```bash
# Cross-platform build
sh scripts/build.sh

# The build script creates static binaries in release/linux/${GOARCH}/
```

## Architecture

### Core Components

**cmd/drone-server** - Main server application entry point

**Core Business Logic:**
- `core/` - Core business logic and interfaces
- `service/` - Business services (user, repo, build management)
- `trigger/` - Pipeline trigger mechanisms (cron, DAG)

**API Layer:**
- `handler/api/` - REST API endpoints  
- `handler/web/` - Web UI serving
- `handler/health/` - Health check endpoints

**Data Layer:**
- `store/` - Data access layer with multiple backends
- `store/shared/` - Common database utilities and migrations

**Service Modules:**
- `service/token/` - Authentication token management
- `service/commit/` - Git commit handling
- `service/license/` - License validation
- `service/linker/` - Repository linking
- `service/canceler/` - Build cancellation
- `service/org/` - Organization management
- `service/transfer/` - Data transfer utilities
- `service/content/` - Content management
- `service/user/` - User management

### Database Support

The platform supports multiple database backends:
- **SQLite** (default, suitable for development)
- **MySQL** (production)
- **PostgreSQL** (production)

Database configuration through environment variables:
- `DRONE_DATABASE_DRIVER` (sqlite3, mysql, postgres)
- `DRONE_DATABASE_DATASOURCE` (connection string)

### Docker Architecture

Multiple Docker images for different architectures:
- `docker/Dockerfile.server.linux.amd64`
- `docker/Dockerfile.server.linux.arm64`  
- `docker/Dockerfile.server.linux.arm`

## Development Workflow

### Local Development Setup
```bash
# Install Go dependencies
go mod download

# Run with SQLite (development)
go run cmd/drone-server/main.go

# Run with environment configuration
export DRONE_GITHUB_CLIENT_ID=...
export DRONE_GITHUB_CLIENT_SECRET=...
go run cmd/drone-server/main.go
```

### Testing Strategy
```bash
# Unit tests
go test ./...

# Race condition testing
go test -race ./...

# Integration testing with MySQL
task test-mysql

# Integration testing with PostgreSQL  
task test-postgres

# Specific store tests
go test -count=1 github.com/drone/drone/store/batch
go test -count=1 github.com/drone/drone/store/repos
go test -count=1 github.com/drone/drone/store/user
```

### Docker Development
```bash
# Build development image
task docker

# Run MySQL for testing
docker run -p 3306:3306 \
  --env MYSQL_DATABASE=test \
  --env MYSQL_ALLOW_EMPTY_PASSWORD=yes \
  --name mysql --detach --rm mysql:5.7

# Run PostgreSQL for testing
docker run -p 5432:5432 \
  --env POSTGRES_PASSWORD=postgres \
  --env POSTGRES_USER=postgres \
  --name postgres --detach --rm postgres:9-alpine
```

## Key Dependencies

**Core Dependencies:**
- `github.com/drone/drone-go` - Drone Go client
- `github.com/drone/drone-runtime` - Runtime execution
- `github.com/drone/drone-yaml` - YAML pipeline parsing
- `github.com/go-chi/chi` - HTTP router
- `github.com/jmoiron/sqlx` - Database toolkit
- `github.com/sirupsen/logrus` - Structured logging

**Database Drivers:**
- `github.com/lib/pq` - PostgreSQL driver  
- `github.com/go-sql-driver/mysql` - MySQL driver
- `github.com/mattn/go-sqlite3` - SQLite driver

**Cloud Storage:**
- `github.com/aws/aws-sdk-go` - AWS S3 integration
- `github.com/Azure/azure-storage-blob-go` - Azure Blob storage

## CI/CD Pipeline

The project uses Drone CI itself for continuous integration:

**Pipeline Stages (.drone.yml):**
1. **Test** - Run unit tests and build validation
2. **Build** - Cross-platform binary compilation  
3. **Publish** - Docker image building and registry push
4. **Manifest** - Multi-architecture manifest creation

**Platform Support:**
- linux/amd64 (Docker pipeline)
- linux/arm64 (VM pipeline)
- Automatic Docker tagging and publishing

## Configuration

**Environment Variables:**
- `DRONE_GITHUB_CLIENT_ID` - GitHub OAuth client ID
- `DRONE_GITHUB_CLIENT_SECRET` - GitHub OAuth secret  
- `DRONE_DATABASE_DRIVER` - Database backend selection
- `DRONE_DATABASE_DATASOURCE` - Database connection string

**Build Tags:**
- `oss` - Open source build without enterprise features
- `nolimit` - Remove usage limitations

## Testing Database Integration

**MySQL Test Setup:**
```bash
# Automatic setup via task
task test-mysql

# Manual setup
docker run -p 3306:3306 \
  --env MYSQL_DATABASE=test \
  --env MYSQL_ALLOW_EMPTY_PASSWORD=yes \
  --name mysql --detach --rm mysql:5.7
```

**PostgreSQL Test Setup:**
```bash
# Automatic setup via task  
task test-postgres

# Manual setup
docker run -p 5432:5432 \
  --env POSTGRES_PASSWORD=postgres \
  --name postgres --detach --rm postgres:9-alpine
```