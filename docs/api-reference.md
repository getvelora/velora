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
| `path` | string | yes      | Absolute **container path** within `VELORA_MEDIA_DIR`. Must be unique. |

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

## `POST /api/libraries/{id}/scan`

Synchronously discovers supported video files under a library and atomically reconciles its persisted inventory.
Symlinks are not followed. If traversal fails, the previous inventory remains unchanged.
New, changed, restored, previously failed, and previously unprobed files are inspected with ffprobe. A damaged or
unsupported file remains in inventory with an inspection error while the rest of the scan completes. If ffprobe
cannot be started, the scan fails without changing inventory.

**Response (200)**

```json
{
  "libraryId": 1,
  "discovered": 2,
  "added": 1,
  "updated": 0,
  "unchanged": 1,
  "restored": 0,
  "markedMissing": 0,
  "probed": 1,
  "probeFailed": 0,
  "ignored": 3,
  "startedAt": "2026-06-09T12:00:00Z",
  "completedAt": "2026-06-09T12:00:00.125Z"
}
```

Supported extensions are matched case-insensitively: `.mkv`, `.mp4`, `.m4v`, `.mov`, `.avi`, `.webm`, `.mpeg`,
`.mpg`, `.ts`, `.m2ts`, and `.wmv`.

| Code | Meaning                                                        |
|------|----------------------------------------------------------------|
| 200  | Scan completed and inventory was reconciled.                   |
| 404  | The library does not exist.                                    |
| 409  | Another scan of the same library is already running.           |
| 500  | The path is unavailable/unsafe, traversal failed, or save failed. |

## `GET /api/libraries/{id}/files`

Returns all persisted files for a library, ordered by relative path. Missing files remain visible.

**Response (200)**

```json
[
  {
    "id": 12,
    "libraryId": 1,
    "path": "movies/Example.mkv",
    "size": 734003200,
    "modifiedAt": "2026-06-01T10:30:00Z",
    "status": "available",
    "firstSeenAt": "2026-06-09T12:00:00Z",
    "lastSeenAt": "2026-06-09T12:05:00Z",
    "missingAt": null,
    "inspection": {
      "status": "ready",
      "error": null,
      "probedAt": "2026-06-09T12:00:00Z",
      "format": {
        "name": "matroska,webm",
        "longName": "Matroska / WebM",
        "durationMs": 7265123,
        "bitRate": 18432000
      },
      "streams": [
        {
          "index": 0,
          "type": "video",
          "codecName": "hevc",
          "codecLongName": "H.265 / HEVC",
          "profile": "Main 10",
          "level": 153,
          "language": "eng",
          "title": "Main video",
          "default": true,
          "forced": false,
          "width": 3840,
          "height": 2160,
          "pixelFormat": "yuv420p10le",
          "bitDepth": 10,
          "frameRate": "24000/1001",
          "colorRange": "tv",
          "colorSpace": "bt2020nc",
          "colorTransfer": "smpte2084",
          "colorPrimaries": "bt2020",
          "sampleRate": 0,
          "channels": 0,
          "channelLayout": ""
        }
      ]
    }
  }
]
```

Returns `404` when the library does not exist and `500` when inventory cannot be loaded.
Inspection status is `unprobed`, `ready`, or `error`. Unprobed and failed inspections return `format: null` and an
empty `streams` array. Failed inspections include a sanitized, length-limited `error`. Only video, audio, and subtitle
streams are returned; attachment and data streams are ignored.

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
- Background scanner control endpoints (start, status, cancel)
- Playback session lifecycle endpoints
