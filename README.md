# Image Service (Go)

A lightweight, high-performance microservice for uploading, processing and serving images.  
The service is written in Go, uses `libvips` (via `govips`) for image manipulation, persists files to S3 (or any S3-compatible storage) and stores metadata in PostgreSQL.  
Phase 1 implements **user images**; organisation and product images will follow the same pattern.

---

## 1 – Project Overview

| Feature | Status | Notes |
|---------|--------|-------|
| User image upload / delete / fetch | ✅ | `/v1/me/image`, `/v1/users/{uid}/image` |
| Organisation & product images | ⏳ | Planned next phases |
| Config-driven sizes | ✅ | Defined in `config/images.yaml` |
| JWT auth middleware | ✅ | RS256 / HS256 |
| S3 adapter | ✅ | Mocked in tests |
| CI / Docker / Makefile | ✅ | GitHub Actions builds & tests |

---

## 2 – Build & Run

### Requirements
* Go 1.21+
* libvips (`sudo apt-get install libvips-dev`) for local builds
* PostgreSQL 14+
* An S3 bucket or MinIO for storage

### Quick start (local)

```bash
# Clone the repo
git clone https://github.com/antonrybalko/image-service-go
cd image-service-go

# Install deps & build binary
make deps build

# Start the service with default config
PORT=8080 go run ./cmd/server
```

### Docker

```bash
make docker-build              # builds antonrybalko/image-service-go:latest
docker run -p 8080:8080 \
  -e PORT=8080 \
  -e S3_BUCKET=images \
  antonrybalko/image-service-go:latest
```

---

## 3 – API End-points (Phase 1)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| **PUT**  | `/v1/me/image`            | JWT | Upload / replace caller’s image |
| **GET**  | `/v1/me/image`            | JWT | Fetch caller’s image metadata |
| **DELETE** | `/v1/me/image`          | JWT | Delete caller’s image |
| **GET**  | `/v1/users/{userUid}/image` | Public | Public metadata lookup |

### Example Response

```json
{
  "userGuid"  : "2d77ab5c-3c55-4f4e-9db1-df3d2f89c12f",
  "imageGuid" : "a4c47f0d-b604-4da6-9e65-21ddf5b2b279",
  "smallUrl"  : "https://cdn.example.com/images/user/2d77ab5c/small.jpg",
  "mediumUrl" : "…/medium.jpg",
  "largeUrl"  : "…/large.jpg",
  "updatedAt" : "2025-06-06T12:34:56Z"
}
```

*All write operations are idempotent when the `Idempotency-Key` header is supplied.*

---

## 4 – Configuration

The service reads **environment variables** (12-factor) and an optional **YAML** size descriptor.

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP port |
| `ENVIRONMENT` | `development` | `production` enables zap production logger |
| **Postgres** |||
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` | | Connection settings |
| **S3** |||
| `S3_REGION` | `us-east-1` | |
| `S3_BUCKET` | `images` | Bucket name |
| `S3_ENDPOINT` | _(empty)_ | Point to MinIO for local dev |
| `S3_CDN_BASE_URL` | _(empty)_ | If set, returned URLs are rewritten to use the CDN |
| `S3_USE_PATH_STYLE` | `false` | Needed for MinIO/localstack |
| **JWT** |||
| `JWT_ALGORITHM` | `RS256` | `HS256` also supported |
| `JWT_PUBLIC_KEY_URL` / `JWT_SECRET` | | Key material |

`config/images.yaml` defines the allowed image types & their variants:

```yaml
images:
  - name: user
    sizes:
      small:  { width: 50,  height: 50 }
      medium: { width: 100, height: 100 }
      large:  { width: 800, height: 800 }
```

---

## 5 – Development Guide

### Project layout

```
cmd/server          ─ main entrypoint
internal/config     ─ env + YAML loader
internal/api        ─ HTTP handlers, routers
internal/auth       ─ JWT middleware
internal/processor  ─ image resizing logic (govips)
internal/storage    ─ S3 adapter
internal/repository ─ Postgres access
internal/domain     ─ business entities
internal/mocks      ─ generated test doubles
```

### Common tasks

| Command | Action |
|---------|--------|
| `make build` | Compile binary to `./bin` |
| `make run` | Run service (reads local env) |
| `make test` | Run tests + coverage |
| `make docker-build` | Build Docker image |
| `make lint` | Run `golangci-lint` |

### Testing

* S3 interactions are **mocked** with Go interfaces & `testify/mock`.
* Run `make test` – unit tests cover config loading, handlers and storage logic.
* Integration tests against real MinIO/Postgres can be added via Docker Compose.

### Contributing

1. Fork → feature branch → PR to `develop`.
2. Ensure `make lint test` passes.
3. Describe context & motivation in the PR body.

### Future Road-map

* Organisation & product endpoints with ownership validation
* Kafka event emission (`ImageUploaded`, `ImageDeleted`)
* Metrics & OpenTelemetry tracing
* Smart caching headers & CloudFront integration

---

© 2025 Anton Rybalko – MIT License
