# API Reference

All endpoints live under `/api/`. Responses are JSON; the server sets `Content-Type: application/json` on both success
and error paths.

## `GET /api/health`

Reports overall server status and the active database driver. Cheap; suitable as a Docker/Kubernetes/Unraid healthcheck.

**Response body**

```json
{
  "status": "ok",
  "database": "ok",
  "databaseDriver": "sqlite"
}
```

| Field            | Type   | Notes                                                          |
|------------------|--------|----------------------------------------------------------------|
| `status`         | string | `"ok"` when healthy; `"degraded"` when any check fails         |
| `database`       | string | `"ok"` when the database ping succeeds; `"error"` otherwise    |
| `databaseDriver` | string | `"sqlite"` or `"postgres"`; echoes `VELORA_DATABASE_DRIVER`    |

**Status codes**

| Code | Meaning                                                                    |
|------|----------------------------------------------------------------------------|
| 200  | All checks passing.                                                        |
| 503  | At least one check failing. Body still returned with per-component detail. |

## `GET /api/libraries`

Returns all configured media libraries, ordered by insertion.

**Response (200)**

```json
[
  {
    "id": 1,
    "name": "Movies",
    "path": "/media/movies",
    "createdAt": "2026-05-21T03:30:18.123456789Z",
    "updatedAt": "2026-05-21T03:30:18.123456789Z"
  }
]
```

Always returns an array (`[]` when no libraries exist), never `null`. Timestamps are RFC 3339 nanosecond UTC.

## `POST /api/libraries`

Create a new library.

**Request body**

```json
{
  "name": "Movies",
  "path": "/media/movies"
}
```

| Field  | Type   | Required | Notes                                                       |
|--------|--------|----------|-------------------------------------------------------------|
| `name` | string | yes      | Display name. Not required to be unique.                    |
| `path` | string | yes      | **Container path** (e.g. `/media/movies`). Must be unique.  |

**Status codes**

| Code | Meaning                                                                          |
|------|----------------------------------------------------------------------------------|
| 201  | Created. Response body is the new library with `id` and timestamps populated.    |
| 400  | Missing/empty `name` or `path`, or invalid JSON.                                 |
| 409  | A library with this `path` already exists. Response body has an `error` field.  |

**Error body shape (400, 409, 500)**

```json
{"error": "library with this path already exists"}
```

## Method-not-allowed semantics

Methods other than the ones documented return `405 Method Not Allowed` with an `Allow` header listing the supported
methods. For example, `DELETE /api/libraries` returns:

```
HTTP/1.1 405 Method Not Allowed
Allow: GET, POST
Content-Type: application/json

{"error": "method not allowed"}
```

## Planned endpoints

Coming in upcoming milestones:

- `GET /api/libraries/{id}` — fetch a single library
- `PUT /api/libraries/{id}` — update name and path
- `DELETE /api/libraries/{id}` — remove a library
- `GET /api/libraries/{id}/files` — list scanned files in a library
- Scanner control endpoints (start, status, cancel)
- Playback session lifecycle endpoints
