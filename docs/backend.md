# FindMe Backend

## Architecture

The backend is a Go HTTP service built with Gin. It follows a module-oriented handler/service/repository architecture:

```text
cmd/api/main.go           dependency wiring and routes
internal/config           environment parsing
internal/database         GORM/PostgreSQL connection and migration runner
internal/redis            Redis connection
internal/middlewares      JWT, CORS, and rate limiting
internal/validator        coordinates, photo validation, filenames
internal/utils            response envelopes and infrastructure helpers
internal/storage          AWS S3 client and storage service
internal/websocket        hub, clients, handler, event contracts
internal/dto/{module}     request and response contracts by feature
internal/handlers         HTTP transport handlers
internal/services         business logic and repository interfaces
internal/repositories     GORM persistence and transactions
migrations                ordered PostgreSQL SQL migrations
```

Handlers parse transport input and select HTTP responses. Services depend on
small repository interfaces and own validation, authorization, and
cross-resource business rules. Repositories are the only modules that use
GORM, SQL, and database transactions. Middleware owns cross-cutting HTTP
concerns, while reusable infrastructure helpers live in `internal/utils`.

The project uses a layer-first structure. DTOs remain grouped by feature so
request and response contracts stay easy to locate, while handlers, services,
and repositories are separated into dedicated packages.

## ORM

All PostgreSQL access runs through `*gorm.DB` using
`gorm.io/driver/postgres`. CRUD and locking use GORM's query builder and
transaction APIs. PostgreSQL-specific joins, CTEs, advisory locks, and
`RETURNING` statements use GORM's `Raw`/`Exec` escape hatch inside repository
packages.

Schema changes do not use `AutoMigrate`. Production schema evolution stays
explicit and reviewable through versioned SQL migrations.

## Modules

- `auth`: registration, login, stateless logout response, profile endpoints
- `users`: user persistence and profile rules
- `groups`: membership, limits, invite codes, admin operations
- `locations`: one-time location shares and location photo metadata
- `live_locations`: session lifecycle, latest positions, Redis cache, broadcasts
- `memories`: points, ratings, comments, photos, average recalculation
- `storage`: private S3 object operations and presigned downloads
- `websocket`: authenticated group connections and realtime events

## Database Migration Guide

On API startup, `database.Migrate` uses the GORM connection to:

1. Creates `schema_migrations` if needed.
2. Reads sorted `migrations/*.up.sql`.
3. Runs each unapplied file in a transaction.
4. Records the filename after a successful commit.

Create a migration pair using the next sortable number:

```text
migrations/000002_feature_name.up.sql
migrations/000002_feature_name.down.sql
```

The startup runner applies only `.up.sql`. Down files are supplied for controlled manual rollback; production rollback should be reviewed and executed explicitly.

## REST API

Base path: `/api/v1`

### Public

| Method | Path | Purpose |
| --- | --- | --- |
| POST | `/auth/register` | Create user and return JWT |
| POST | `/auth/login` | Verify password and return JWT |

### Authentication/Profile

| Method | Path | Purpose |
| --- | --- | --- |
| POST | `/auth/logout` | Stateless logout acknowledgement |
| GET | `/auth/me` | Current profile |
| PATCH | `/auth/me` | Update name/avatar URL |

### Groups

| Method | Path |
| --- | --- |
| POST | `/groups` |
| GET | `/groups` |
| GET | `/groups/:groupId` |
| PATCH | `/groups/:groupId` |
| DELETE | `/groups/:groupId` |
| POST | `/groups/join` |
| POST | `/groups/:groupId/leave` |
| GET | `/groups/:groupId/members` |
| DELETE | `/groups/:groupId/members/:userId` |
| POST | `/groups/:groupId/invite-code/regenerate` |

### Locations

| Method | Path |
| --- | --- |
| POST | `/locations/share` |
| GET | `/groups/:groupId/locations` |
| GET | `/groups/:groupId/locations/latest` |
| POST | `/locations/:locationShareId/photos` |

`POST /locations/share` accepts either `group_id` or `share_to_all: true`, never both.

### Live Location

| Method | Path |
| --- | --- |
| POST | `/groups/:groupId/live-location/start` |
| POST | `/groups/:groupId/live-location/update` |
| POST | `/groups/:groupId/live-location/stop` |
| GET | `/groups/:groupId/live-location/active` |

The update endpoint is limited to 30 requests per minute per authenticated user.

### Memory Points

| Method | Path |
| --- | --- |
| POST | `/groups/:groupId/memory-points` |
| GET | `/groups/:groupId/memory-points` |
| GET | `/memory-points/:memoryPointId` |
| PATCH | `/memory-points/:memoryPointId` |
| DELETE | `/memory-points/:memoryPointId` |
| POST | `/memory-points/:memoryPointId/ratings` |
| POST | `/memory-points/:memoryPointId/comments` |
| GET | `/memory-points/:memoryPointId/comments` |
| POST | `/memory-points/:memoryPointId/photos` |

## WebSocket Events

Connect with:

```text
GET /ws/groups/{groupId}?token={jwt}
```

The server verifies the JWT, Origin, and group membership before upgrading.

Event envelope:

```json
{
  "type": "live_location.updated",
  "data": {}
}
```

Events:

- `live_location.updated`: session/user/group IDs, coordinates, accuracy, heading, speed, and update time
- `live_location.stopped`: session/user/group IDs
- `live_location.expired`: session/user/group IDs

The hub is in-process. For horizontal API scaling, bridge broadcasts through Redis Pub/Sub.

## Authentication Flow

Passwords are hashed with bcrypt's default cost. Login compares the stored hash and issues an HS256 JWT containing the authenticated user UUID in `sub` and `user_id`, with configurable expiry.

Protected handlers obtain identity only through `middlewares.UserID(c)`. Request bodies never provide the acting `user_id`.

Logout is stateless: clients discard the JWT. Add a Redis denylist or refresh-token table later if immediate token revocation is required.

## Authorization Rules

- Any group resource read requires membership.
- Group updates, member removal, and invite regeneration require admin role.
- Only the creator may delete a group.
- The creator cannot leave; they delete the group instead.
- A user can join at most five groups.
- A group can contain at most ten users.
- Live sessions require membership and one active session globally per user.
- Memory editing/deletion requires point ownership or group admin role.
- Ratings, comments, and photos require current group membership.

## S3 Upload Flow

1. Gin parses multipart form field `photos`.
2. Service verifies membership and existing photo count.
3. Every file is validated before the first upload.
4. Allowed MIME types/extensions: JPEG/JPG, PNG, WebP.
5. Maximum size: 5 MB each; maximum five photos per resource.
6. The service uploads to private S3.
7. Repository inserts metadata.
8. A database failure triggers best-effort S3 cleanup.
9. Responses generate presigned GET URLs using the configured expiry.

No local filesystem is used for permanent photos.

## Redis Usage

- `rate:live:{userId}`: fixed-window live update counter
- `live:group:{groupId}:user:{userId}`: temporary serialized latest position

The position key expires at session expiry. PostgreSQL stores the authoritative session and latest position.

## Error Response Format

All API JSON uses a consistent envelope.

Success:

```json
{
  "success": true,
  "data": {}
}
```

Error:

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "latitude must be between -90 and 90",
    "details": "optional validation detail"
  }
}
```

Typical status codes are 200/201 for success, 400 validation, 401 authentication, 403 membership/role, 404 resource/session, 409 business limits, 429 rate limit, and 500 unexpected failures.

## Run Locally

Required Go version: 1.24 or newer.

```bash
cd backend
cp .env.example .env
go mod download
go run ./cmd/api
```

The Go process reads environment variables directly; load `.env` with your shell or dotenv tool. For host-run services, change container hostnames in `DATABASE_URL` and `REDIS_ADDR` to `localhost`.

Verify:

```bash
gofmt -w .
go test ./...
curl http://localhost:8080/health
```
