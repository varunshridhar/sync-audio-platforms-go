# File-by-File Guide (Beginner Friendly)

This document explains files that are hard/impossible to comment inline (mainly strict JSON files), and also gives quick context for key repo files.

## Architecture Diagram (Current MVP)

```text
                               (Browser)
                            User on Web UI
                                  |
                                  | HTTPS (cookie auth)
                                  v
                      +-----------------------------+
                      | Frontend (Next.js on :3000)|
                      | routes/pages + API client   |
                      +-------------+---------------+
                                    |
                                    | HTTP/JSON requests
                                    v
                      +-----------------------------+
                      | Backend API (Go on :8080)  |
                      | middleware + handlers       |
                      +------+------+---------------+
                             |      |
       encrypted tokens/user |      | sync jobs + metadata
       connections            |      |
                             v      v
                +----------------+  +----------------------+
                | Security layer |  | Firestore collections|
                | AES-GCM + HMAC |  | users/accounts/jobs  |
                +----------------+  +----------------------+
                                     ^
                                     |
                           (future async workers)
                           Cloud Tasks -> Worker
                           updates job status
```

## Request Flow Examples

### 1) Login Flow

1. User submits email in `frontend/app/login/page.tsx`.
2. Frontend calls `POST /v1/auth/login` via `frontend/lib/api.ts`.
3. Backend validates email, creates/fetches user in Firestore, signs session token.
4. Backend returns user JSON and sets `session` cookie (`HttpOnly`, `SameSite=Strict`).
5. Frontend navigates to dashboard and loads user/providers/jobs.

### 2) Create Sync Job Flow

1. User submits source+destination+playlist in dashboard form.
2. Frontend calls `POST /v1/sync/jobs`.
3. Backend auth middleware verifies session cookie and injects user ID into context.
4. Handler validates providers and writes a new `pending` job to Firestore.
5. Frontend refreshes job list with `GET /v1/sync/jobs`.
6. (Future) Cloud Tasks worker picks the job and updates status to `running/complete/failed`.

## JSON Files (No inline comments supported)

- `frontend/package.json`: Frontend project metadata, dependency list, and npm scripts (`dev`, `build`, `start`, `lint`).
- `frontend/.eslintrc.json`: ESLint rule preset selection (`next/core-web-vitals`) for frontend code quality checks.
- `frontend/tsconfig.json`: TypeScript compiler options for frontend. Controls strict type checking and Next.js type integration.

## Environment Template Files

- `backend/env.example`: Backend env variables template (copy to local env file and fill secrets).
- `frontend/env.local.example`: Frontend env variables template (browser-visible values only).
- `env.docker.example`: Local Docker Compose env template used by `make docker-up-dev`.

## Infra Files

- `docker-compose.yml`: Runs frontend+backend together for local development.
- `Makefile`: Convenience commands for common development and Docker workflows.
- `backend/Dockerfile`: Builds and runs backend in a minimal distroless runtime image.
- `frontend/Dockerfile`: Builds and runs frontend in production mode.
- `.gitignore`: Prevents local artifacts and secrets from being committed.
- `.gcloudignore`: Prevents local secrets/artifacts from being uploaded during GCP deploy/build.

## Backend Flow Files

- `backend/cmd/api/main.go`: Process entrypoint, config load, HTTP server lifecycle.
- `backend/internal/app/*`: Route registration and request handlers.
- `backend/internal/httpx/*`: Middleware and HTTP response helper utilities.
- `backend/internal/store/*`: DB abstraction and Firestore implementation.
- `backend/internal/security/*`: Token encryption and session signing/verification.
- `backend/internal/domain/types.go`: Core data models used across the backend.

## Frontend Flow Files

- `frontend/app/*`: Pages/layout/styles for home, login, and dashboard.
- `frontend/lib/api.ts`: Browser API client functions used by UI pages.
- `frontend/middleware.ts`: Frontend security headers middleware.
- `frontend/next.config.mjs`: Next.js framework settings.

