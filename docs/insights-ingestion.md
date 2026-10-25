# Insights event ingestion

Insights accepts events from official Sentry SDKs. The ingestion endpoints
follow the Sentry wire protocol and are intentionally not part of the OpenAPI
specification (they are not session authenticated and have no JSON API
contract). The admin API (projects, issues, events, key rotation) is
documented in OpenAPI under the `insights` tag.

## Endpoints

| Method | Path                          | Body                                   |
|--------|-------------------------------|----------------------------------------|
| POST   | `/api/{project_id}/envelope/` | Sentry envelope (what current SDKs use) |
| POST   | `/api/{project_id}/store/`    | Single JSON event (legacy SDKs)         |

`{project_id}` is the numeric id of an Insights project. A trailing slash is
optional. `OPTIONS` preflight requests are answered with permissive CORS
(any origin, no credentials), so browser SDKs work cross origin.

Authentication is the DSN public key, sent as one of:

- the `sentry_key` query parameter,
- the `X-Sentry-Auth` header (`Sentry sentry_key=<key>, sentry_version=7, ...`),
- `Authorization: Sentry sentry_key=<key>`.

Request bodies may be compressed (`Content-Encoding: gzip` or `deflate`).
Limits: 1 MiB compressed, 20 MiB decompressed, at most 100 envelope items.

Responses:

| Status | Meaning                                                                   |
|--------|---------------------------------------------------------------------------|
| 200    | Accepted, body `{"id": "<event id>"}`                                     |
| 400    | Malformed envelope or event                                               |
| 403    | Unknown project, missing/invalid key, or key does not match the project   |
| 405    | Method other than POST                                                    |
| 413    | Payload too large or too many items                                       |
| 429    | Project rate limit hit; honour `Retry-After` / `X-Sentry-Rate-Limits`     |
| 500    | Internal error                                                            |

Errors use the Sentry style body `{"detail": "..."}`.

## Configure an SDK

1. Create a project for the repository (needs repository edit permission):

   ```sh
   curl -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
     -d '{"identifier": "my-service"}' \
     https://<host>/api/v1/repos/<repo_ref>/insights/projects
   ```

   The response contains a `dsn` field such as
   `https://<public_key>@<host>/<project_id>`.

2. Pass that DSN to the SDK, unchanged. Examples:

   ```js
   // JavaScript
   Sentry.init({ dsn: "https://<public_key>@<host>/<project_id>" });
   ```

   ```python
   # Python
   sentry_sdk.init(dsn="https://<public_key>@<host>/<project_id>")
   ```

   ```go
   // Go
   sentry.Init(sentry.ClientOptions{Dsn: "https://<public_key>@<host>/<project_id>"})
   ```

   Set `release` (and optionally `environment`) in the SDK options; issues can
   be filtered by release and resolved "in release".

3. Trigger an error and check `GET /api/v1/repos/<repo_ref>/insights/issues?project=<project_id>`.

## Project settings and keys

`PATCH /repos/{repo_ref}/insights/projects/{project_id}` toggles `enabled` and
sets `rate_limit_per_minute` (default 600, maximum 100000) and `sample_rate`
(0 to 1). `POST .../rotate-key` issues a new public key and returns the new
DSN; the old key stops working immediately, so update deployed SDK
configuration afterwards. The public key is not a secret in browser setups,
but treat the project rate limit as the abuse control.
