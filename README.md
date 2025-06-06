# Image Service Go

Fast, configurable micro-service written in Go for **uploading, processing and serving images** for users, organisations (and in the future – products).  
It isolates all image concerns from the `user-service`, giving cleaner boundaries and independent scalability.

---

## 1. Project Overview
* Stateless HTTP API that:
  * accepts an image
  * resizes it to three configured variants (small, medium, large)
  * stores objects in S3-compatible storage
  * returns public URLs + metadata
* Authenticated _“/v1/me”_ routes for current user, optional organisation routes, public read-only routes.
* Configuration-driven image types – add new kinds by editing `config/images.yaml`.
* Built with libvips (via Govips) for high-performance resizing.

---

## 2. Getting Started

```bash
# clone
git clone https://github.com/antonrybalko/image-service-go.git
cd image-service-go

# configure environment
cp .env.example .env      # edit values

# run locally
make deps
make run                  # http://localhost:8080/health
```

Docker:

```bash
make docker-build
docker run --env-file .env -p 8080:8080 image-service-go:latest
```

---

## 3. Configuration

| Variable                         | Default          | Description                              |
|---------------------------------|------------------|------------------------------------------|
| ENVIRONMENT                      | development      | `development` \| `production`            |
| PORT                             | 8080             | HTTP port                                |
| S3_REGION/BUCKET/\*              | —                | S3 credentials & bucket                  |
| JWT_PUBLIC_KEY_URL / JWT_SECRET  | —                | One of them depending on algorithm       |
| IMAGE_CONFIG_PATH                | config/images.yaml | YAML with image type & size definitions |

See `internal/config/config.go` for full list and defaults.

`config/images.yaml` sample:

```yaml
images:
  - name: user
    sizes:
      small:  { width: 50,  height: 50 }
      medium: { width: 100, height: 100 }
      large:  { width: 800, height: 800 }
```

---

## 4. Development Workflow

| Task                       | Command               |
|----------------------------|-----------------------|
| Download deps              | `make deps`           |
| Run locally                | `make run`            |
| Hot reload (air)           | `make dev`            |
| Lint (golangci-lint)       | `make lint`           |
| Unit tests                 | `make test`           |
| Coverage report            | `make coverage`       |
| Build binary               | `make build`          |
| Docker image               | `make docker-build`   |

CI (GitHub Actions) runs `lint`, `test`, `build`, and releases Docker images on git tags.

---

## 5. API Documentation (v1)

### Private (JWT required)

| Method | Path                                         | Purpose                     |
|--------|----------------------------------------------|-----------------------------|
| PUT    | `/v1/me/image`                               | Upload / replace user image |
| GET    | `/v1/me/image`                               | Retrieve user image meta    |
| DEL    | `/v1/me/image`                               | Delete user image           |
| PUT    | `/v1/me/organizations/{orgGuid}/image`       | Upload organisation image   |
| GET    | `/v1/me/organizations/{orgGuid}/image`       | Retrieve organisation image |
| DEL    | `/v1/me/organizations/{orgGuid}/image`       | Delete organisation image   |

### Public

| Method | Path                                   | Purpose                           |
|--------|----------------------------------------|-----------------------------------|
| GET    | `/v1/users/{userGuid}/image`           | Public user image meta            |
| GET    | `/v1/organizations/{orgGuid}/image`    | Public organisation image meta    |

#### Response schema

```json
{
  "userGuid": "uuid",           // or organisationGuid
  "imageGuid": "uuid",
  "smallUrl":  "https://…/small.jpg",
  "mediumUrl": "https://…/medium.jpg",
  "largeUrl":  "https://…/large.jpg",
  "updatedAt": "2025-06-06T12:34:56Z"
}
```

All variants are JPEGs with quality 90; keys are `images/{type}/{ownerGuid}/{imageGuid}/{size}.jpg`.

---

## 6. Technologies Used
* **Go 1.21**
* **chi** – HTTP router & middleware
* **govips** / **libvips** – high-speed image processing
* **zap** – structured logging
* **AWS SDK v2** – S3 client (mocked in tests)
* **Viper** – configuration loader
* **Docker** – containerisation
* **GitHub Actions** – CI/CD

---

## 7. Project Structure

```
.
├── cmd/
│   └── server/         # main.go – entrypoint
├── internal/
│   ├── api/            # HTTP handlers & routing
│   ├── auth/           # JWT middleware (future)
│   ├── config/         # config loader
│   ├── processor/      # resize logic
│   ├── storage/        # S3 adapter (+ mock)
│   ├── repository/     # PostgreSQL access (future)
│   └── domain/         # business entities / DTOs
├── config/             # images.yaml
├── tests/              # integration tests & mocks
└── ...
```

---

## 8. Testing Approach

* **Unit tests** (`go test ./...`) using `stretchr/testify`.
* **Mocked S3** – in-memory implementation behind `storage.Interface` ensures tests run without AWS.
* **Config tests** – verify defaults and env overrides.
* **HTTP handler tests** – request/response scenarios via `httptest`.
* **Coverage gate** in CI; `make coverage` produces HTML report.

Future phases will add integration tests with Postgres & localstack S3 in Docker compose.
