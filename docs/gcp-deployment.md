# GCP Deployment (Low-Cost Secure Path)

## Services

- `frontend` on Cloud Run (public).
- `backend` on Cloud Run (public, CORS restricted to frontend URL).
- Firestore Native mode in `asia-south1`.
- Cloud Tasks queue for async sync workers.
- Secret Manager for all secrets.

## Recommended Runtime Setup

- Run frontend/backend in separate Cloud Run services with minimum instances `0`.
- Use service-to-service auth for internal worker callbacks.
- Configure custom domain and HTTPS for frontend.
- Only expose backend endpoints needed by frontend and provider webhooks.

## Secrets

Store and mount these from Secret Manager:

- `SESSION_HMAC_KEY` (minimum 32 chars).
- `TOKEN_ENCRYPTION_KEY` (base64-encoded 32-byte key).
- Provider client IDs/secrets.

## Firestore Security

- Keep Firestore private to backend service account (no direct browser access).
- Assign minimum IAM role: `roles/datastore.user` to backend runtime account only.
- Use separate project/environment per stage (`dev`, `staging`, `prod`) if possible.

## Hardening Checklist

- Restrict CORS origin to your exact frontend domain.
- Enable Cloud Armor for backend with basic WAF rules.
- Turn on Cloud Audit Logs, Cloud Logging retention policy.
- Set alerting for spikes in 4xx/5xx, latency, and queue depth.
- Rotate secrets regularly and immediately on incident.

