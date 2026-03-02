# Sync Audio Platforms

Sync playlists across providers with a secure Go backend and Next.js frontend.

## Documentation Index

Use these docs in order:

1. Local development (no cloud account required): `docs/local-development.md`
2. GCP production deployment (Cloud Run + Firestore): `docs/deployment-gcp.md`
3. AWS production deployment (ECS Fargate + EFS SQLite): `docs/deployment-aws.md`
4. Codebase walkthrough: `docs/file-by-file-guide.md`

## API Docs Endpoints

- Interactive docs: `GET /v1/docs`
- OpenAPI spec: `GET /v1/docs/openapi.yaml`

## Repository Layout

- `backend`: Go API for auth/session, provider connections, and sync jobs.
- `frontend`: Next.js app for landing, login/request-access, dashboard, and admin approvals.
- `docs`: runbooks and architecture docs.

## Quick Commands

```bash
cd /home/varun/personal/sync-audio-platforms-go
make help
```

Useful targets:

- `make backend-run` (auto-loads `backend/env` when present)
- `make backend-run-sqlite`
- `make backend-test`
- `make frontend-dev`
- `make e2e-local-sqlite`
- `make docker-up-dev`
- `make docker-down-dev`

