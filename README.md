# gateway

Extensible API gateway in Go. Reads backends and routes from YAML,
proxies requests via `httputil.ReverseProxy`, applies per-backend
timeouts and custom headers. Sits in front of multiple backend services.

## Progress

### Done

- [x] Env config loading (`godotenv` + `os.LookupEnv`)
- [x] Structured logging (`log/slog`)
- [x] YAML config for backends, routes, defaults
- [x] Defaults merge (timeout, health_path, retries, headers)
- [x] Reverse proxy with `Rewrite` (not deprecated `Director`)
- [x] Header injection via `Rewrite` + `SetXForwarded`
- [x] Per-backend timeout (`context.WithTimeout`)
- [x] Hardcoded `30s` timeout fallback
- [x] Global per-request timeout middleware (10s)
- [x] Panic recovery middleware (catches panics, returns 500)
- [x] Request logging middleware (method, path, status, latency)
- [x] Uniform error envelope `{"error":{"code":"...","message":"..."}}`
- [x] `NoRoute` handler (404 for unmatched paths)
- [x] Proxy `ErrorHandler` (uniform 502 when backend unreachable)
- [x] Backend `name` + `url` validation at startup

### Planned

- [ ] Graceful shutdown — Phase 5
  - Replace `router.Run` with `http.Server`
  - Add `ReadTimeout`, `WriteTimeout`, `ShutdownTimeout` to config + `.env.example`
  - `signal.Notify` for SIGINT/SIGTERM; `server.Shutdown(ctx)` with deadline
  - Log shutdown progress; exit 0 on clean, 1 on timeout
  - Filter `http.ErrServerClosed` from serve error
- [ ] Dockerization (multi-stage Dockerfiles + compose) — Phase 6
  - `gateway/Dockerfile` with `dev` + `prod` targets
  - `gateway/.dockerignore`
  - `docker-compose.yml` at repo root, gateway depends on music
- [ ] Postgres connection pool + migrations — Phase 7
  - Add `pgx`, `pgxpool`, `golang-migrate`
  - Add `DATABASE_URL` to config (required, errors if missing)
  - `migrations/000001_create_users.up.sql` + `.down.sql`
  - `internal/store/db.go` (pool creation + migration runner, handle `ErrNoChange`)
  - Open pool + run migrations at startup, close pool on shutdown
  - Postgres service in `docker-compose.yml`
- [ ] User registration with argon2id — Phase 8
  - Add `golang.org/x/crypto/argon2`
  - `internal/auth/password.go` (hash + verify in PHC string format)
  - `internal/store/users.go` (`UserStore`: `Create`, `GetByEmail`)
  - `POST /register`; validate input, hash password, store user
  - Unique-constraint violation on email -> 409 `email_exists`
- [ ] Login + JWT access tokens — Phase 9
  - Add `golang-jwt/jwt/v5`
  - Add `JWT_SECRET` (required, no default) + `JWT_ACCESS_TTL` to config
  - `internal/auth/jwt.go` (HS256, claims: sub, iat, exp, jti, iss)
  - `POST /login`; same 401 `invalid_credentials` for wrong password + unknown email
- [ ] Refresh tokens with Redis — Phase 10
  - Add `redis/go-redis/v9`
  - Add `REDIS_URL` + `JWT_REFRESH_TTL` to config
  - `internal/store/tokens.go` (opaque refresh tokens, TTL = refresh lifetime)
  - `POST /refresh` with token rotation (delete old, issue new)
  - `POST /login` issues both access + refresh tokens
  - Redis service in `docker-compose.yml`
- [ ] Auth middleware — Phase 11
  - `internal/auth/verify.go` (parse + verify JWT)
  - `internal/middleware/auth.go` (Bearer extraction, verify, set user ID on context)
  - Public group: `/healthz`, `/register`, `/login`, `/refresh`
  - Protected group: proxy routes
  - 401 `missing_token` / `invalid_token` for missing or bad tokens
- [ ] Token revocation + logout — Phase 12
  - `BlocklistAccess(ctx, jti, ttl)` + `IsBlocked(ctx, jti)` in token store
  - Blocklist TTL = access token remaining lifetime
  - Auth middleware checks blocklist after verifying JWT
  - `POST /logout`; blocklist access jti, delete refresh token, idempotent
- [ ] Active health checking (`health_path` stored, not wired)
- [ ] Retry logic (`retries` stored, not wired)

## Configuration

### Environment variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GATEWAY_ADDR` | Listen address | `:9999` |
| `LOG_LEVEL` | slog level (`debug`/`info`/`warn`/`error`) | `info` |
| `CONFIG_FILE` | Path to YAML config | `config.yaml` |
| `MUSIC_BACKEND_URL` | Music backend base URL | `http://localhost:8080` |

Copy `.env.example` to `.env`. The `.env` file is gitignored.

### YAML config (`config.yaml`)

The config file defines a `defaults` block, a list of `backends`, and a
list of `routes`.

```yaml
defaults:
  timeout: 10s
  health_path: /healthz
  retries: 0
  headers:
    X-Forwarded-By: sim-gateway

backends:
  - name: music
    url: http://localhost:8080
    timeout: 5s
    headers:
      X-Forwarded-Service: music

routes:
  - method: GET
    path: /albums
    backend: music
  - method: GET
    path: /album/:id
    backend: music
  - method: POST
    path: /album
    backend: music
```

- **Defaults merge:** backends inherit from `defaults` when a field is
  omitted. Headers extend defaults, do not replace them.
- **Timeout fallback:** empty timeout falls back to hardcoded `30s`.
  Invalid timeout string (`"abc"`) is a startup error.
- **Validation:** `name` and `url` required. Missing either is a startup
  error.

## Architecture

```
gateway/
├── cmd/gateway/main.go              # entry point
├── internal/
│   ├── config/config.go             # env loading
│   ├── logger/logger.go             # slog setup
│   ├── routing/config.go            # YAML structs + loader + defaults merge
│   ├── routing/proxy.go             # proxy table (name -> proxy + timeout + headers)
│   ├── middleware/logging.go        # per-request log line
│   ├── middleware/recovery.go       # panic recovery -> 500
│   ├── middleware/timeout.go        # global per-request deadline
│   └── httperr/respond.go           # uniform error envelope
├── config.yaml                      # backends + routes + defaults
├── .env.example
├── go.mod
└── go.sum
```

## Running

```bash
# start a backend (e.g. music on localhost:8080), then:
go run ./cmd/gateway
```

```bash
curl -i localhost:9000/healthz        # 200 health check
curl -i localhost:9000/albums         # 200 proxied response
curl -i localhost:9000/nonexistent    # 404 unmatched route
# stop the backend, then:
curl -i localhost:9000/albums         # 502 uniform error envelope
```
