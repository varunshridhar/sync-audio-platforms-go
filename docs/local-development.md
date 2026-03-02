# Local Development Guide

## 1) Generate required secrets

```bash
openssl rand -hex 32
openssl rand -base64 32
```

Use:

- `SESSION_HMAC_KEY=<hex-output>`
- `TOKEN_ENCRYPTION_KEY=<base64-output>`

## 2) Backend local setup (SQLite, no GCP needed)

```bash
cd /home/varun/personal/sync-audio-platforms-go/backend
cp env.example env
```

Edit `backend/env`:

- `STORE_PROVIDER=sqlite`
- `SQLITE_PATH=/tmp/sync-audio-platforms.db`
- set valid `SESSION_HMAC_KEY`
- set valid `TOKEN_ENCRYPTION_KEY`
- set optional `SIGNUP_ACCESS_CODES`

Run:

```bash
cd /home/varun/personal/sync-audio-platforms-go
make backend-run
```

## 3) Frontend local setup

```bash
cd /home/varun/personal/sync-audio-platforms-go/frontend
cp env.local.example .env.local
npm install
npm run dev
```

If backend is on default local port, use:

```env
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
```

## 4) Local health checks

```bash
curl http://localhost:8080/v1/health
curl http://localhost:8080/v1/docs/openapi.yaml | head -n 5
```

## 5) Automated local e2e connectivity test

```bash
cd /home/varun/personal/sync-audio-platforms-go
make e2e-local-sqlite
```

This target starts isolated backend+frontend instances, creates a test user via login API, and verifies authenticated `/v1/me`.
