# GCP Deployment Runbook (End-to-End)

This guide deploys:

- `backend` -> Cloud Run
- `frontend` -> Cloud Run
- Firestore (for `STORE_PROVIDER=firestore`)
- Secret Manager for runtime secrets
- Artifact Registry for container images

## 1) Prerequisites

- `gcloud` CLI installed and authenticated.
- Billing-enabled GCP project.
- Domain optional (you can start with Cloud Run URLs).

Set shell variables once:

```bash
export PROJECT_ID="your-gcp-project-id"
export REGION="asia-south1"
export REPO="sync-audio"
export BACKEND_SERVICE="sync-audio-backend"
export FRONTEND_SERVICE="sync-audio-frontend"
```

## 2) Enable required APIs

```bash
gcloud config set project "$PROJECT_ID"
gcloud services enable \
  run.googleapis.com \
  artifactregistry.googleapis.com \
  firestore.googleapis.com \
  secretmanager.googleapis.com \
  cloudbuild.googleapis.com
```

## 3) Create Artifact Registry repo

```bash
gcloud artifacts repositories create "$REPO" \
  --repository-format=docker \
  --location="$REGION" \
  --description="Sync Audio container images"
```

Image URLs you will use:

```bash
export BACKEND_IMAGE="$REGION-docker.pkg.dev/$PROJECT_ID/$REPO/backend:latest"
export FRONTEND_IMAGE="$REGION-docker.pkg.dev/$PROJECT_ID/$REPO/frontend:latest"
```

## 4) Create Firestore (Native mode)

```bash
gcloud firestore databases create --location="$REGION"
```

## 5) Create runtime secrets

Generate values:

```bash
openssl rand -hex 32
openssl rand -base64 32
```

Create secrets:

```bash
printf '%s' '<SESSION_HMAC_KEY_VALUE>' | gcloud secrets create SESSION_HMAC_KEY --data-file=-
printf '%s' '<TOKEN_ENCRYPTION_KEY_VALUE>' | gcloud secrets create TOKEN_ENCRYPTION_KEY --data-file=-
```

Optional (if using Turnstile in prod):

```bash
printf '%s' '<TURNSTILE_SECRET>' | gcloud secrets create TURNSTILE_SECRET_KEY --data-file=-
```

If secret already exists, add new versions:

```bash
printf '%s' '<NEW_VALUE>' | gcloud secrets versions add SESSION_HMAC_KEY --data-file=-
```

## 6) Build and push images

From repo root:

```bash
gcloud auth configure-docker "$REGION-docker.pkg.dev" --quiet

docker build -t "$BACKEND_IMAGE" /home/varun/personal/sync-audio-platforms-go/backend
docker push "$BACKEND_IMAGE"

docker build -t "$FRONTEND_IMAGE" /home/varun/personal/sync-audio-platforms-go/frontend
docker push "$FRONTEND_IMAGE"
```

## 7) Deploy backend Cloud Run service

First deploy backend with placeholder origin, then update after frontend deploy.

```bash
gcloud run deploy "$BACKEND_SERVICE" \
  --image "$BACKEND_IMAGE" \
  --region "$REGION" \
  --platform managed \
  --allow-unauthenticated \
  --port 8080 \
  --set-env-vars APP_ENV=production,STORE_PROVIDER=firestore,FIRESTORE_PROJECT_ID="$PROJECT_ID",DEFAULT_RATE_LIMIT_PER_MIN=120,SIGNUP_RATE_LIMIT_PER_HOUR=10,ACCESS_CODE_MAX_USES=1,ACCESS_CODE_MAX_FAILURES=5,ACCESS_CODE_LOCKOUT_MINUTES=15,ADMIN_EMAILS=dev@example.com,SIGNUP_ACCESS_CODES=Ryanthisisforoouuuuu,ALLOWED_ORIGIN=https://example.com \
  --set-secrets SESSION_HMAC_KEY=SESSION_HMAC_KEY:latest,TOKEN_ENCRYPTION_KEY=TOKEN_ENCRYPTION_KEY:latest
```

If using Turnstile secret:

```bash
gcloud run services update "$BACKEND_SERVICE" \
  --region "$REGION" \
  --set-secrets TURNSTILE_SECRET_KEY=TURNSTILE_SECRET_KEY:latest
```

Get backend URL:

```bash
export BACKEND_URL="$(gcloud run services describe "$BACKEND_SERVICE" --region "$REGION" --format='value(status.url)')"
echo "$BACKEND_URL"
```

## 8) Deploy frontend Cloud Run service

```bash
gcloud run deploy "$FRONTEND_SERVICE" \
  --image "$FRONTEND_IMAGE" \
  --region "$REGION" \
  --platform managed \
  --allow-unauthenticated \
  --port 3000 \
  --set-env-vars NEXT_PUBLIC_API_BASE_URL="$BACKEND_URL",NEXT_PUBLIC_TURNSTILE_SITE_KEY="<TURNSTILE_SITE_KEY_OR_EMPTY>"
```

Get frontend URL:

```bash
export FRONTEND_URL="$(gcloud run services describe "$FRONTEND_SERVICE" --region "$REGION" --format='value(status.url)')"
echo "$FRONTEND_URL"
```

## 9) Update backend CORS origin to real frontend URL

```bash
gcloud run services update "$BACKEND_SERVICE" \
  --region "$REGION" \
  --update-env-vars ALLOWED_ORIGIN="$FRONTEND_URL"
```

## 10) Verify deployment

```bash
curl "$BACKEND_URL/v1/health"
curl "$BACKEND_URL/v1/docs/openapi.yaml" | head -n 5
curl -I "$FRONTEND_URL"
```

Manual browser checks:

- Open frontend URL.
- Request access/login flow.
- Open backend docs at `$BACKEND_URL/v1/docs`.

## 11) Updating releases

```bash
docker build -t "$BACKEND_IMAGE" /home/varun/personal/sync-audio-platforms-go/backend
docker push "$BACKEND_IMAGE"
gcloud run deploy "$BACKEND_SERVICE" --image "$BACKEND_IMAGE" --region "$REGION"

docker build -t "$FRONTEND_IMAGE" /home/varun/personal/sync-audio-platforms-go/frontend
docker push "$FRONTEND_IMAGE"
gcloud run deploy "$FRONTEND_SERVICE" --image "$FRONTEND_IMAGE" --region "$REGION"
```

## 12) Production hardening checklist

- Restrict `ADMIN_EMAILS` to real admin users.
- Configure custom domain + HTTPS for frontend.
- Rotate Secret Manager values periodically.
- Enable Cloud Run metrics/alerts (5xx, latency, request spikes).
- Keep `STORE_PROVIDER=firestore` in GCP production.
