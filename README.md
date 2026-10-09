<p align="center">
  <img src="docs/assets/gorrson-banner.png" alt="Banner">
</p>

# ![FileBernic](internal/api/web/public/icons.svg) gorrson: High-Performance RSS CLI & API Engine

**gorrson** is a modern, high-performance RSS engine written in Go. It provides a robust, private, and multi-user solution for news aggregation that runs anywhere—from small servers to high-end workstations.

## Key Features

1.  **Editorial Experience**: Modern web interface with **Magazine Mode** featuring a dynamic Bento Grid, editorial typography, and intelligent content preview.
2.  **Full Content Scraping**: Extracts the main text from truncated feeds using custom recipes and `go-readability`. Robust charset detection for global support.
3.  **Advanced Filtering**: Rule-based engine with Regex. Actions: `BLOCK`, `HIDE`, `REPLACE`, `HIGHLIGHT`, `IMPORTANT` (Priority) or `NOTIFY` (Bell).
4.  **Fever API Support**: Full compatibility with professional mobile readers like **Reeder**, **Fiery Feeds**, and **Unread**.
5.  **Data Portability**: Native backup/restore of your entire database (SQLite), export/import of OPML feeds, and JSON configuration for settings and filters.
6.  **Privacy First**: Built-in Image Proxy & Cache to protect your identity. Each user has a dedicated, isolated SQLite database.
7.  **Search & Speed**: Ultra-fast full-text search powered by **SQLite FTS5**.

## Performance Benchmarks

Measured on AMD Ryzen 5 5600X with `-benchtime=5s`:

| Operation            | CPU Cores | Latency (ns/op) | Throughput     |
| :------------------- | :-------- | :-------------- | :------------- |
| **Full-Text Search** | 1 CPU     | 4,526,220       | ~221 ops/sec   |
| **List News (Feed)** | 1 CPU     | 186,619         | ~5,358 ops/sec |
| **List News (Feed)** | 4 CPU     | 190,217         | ~5,257 ops/sec |

_Performance remains extremely stable and linear across different CPU configurations._

## Quick Start

### Docker (Recommended)

1.  **Configure** (required):

    ```bash
    cp .env.example .env
    # Edit .env and replace the placeholder with a fresh secret (>= 32 chars):
    openssl rand -base64 32
    ```

    `docker compose` reads `.env` and is **fail-closed**: it aborts if `GORRSON_JWT_SECRET`
    is missing or shorter than 32 characters. Optionally change `GORRSON_PORT_HOST` in the
    same file to publish a different host port (default `7077`). `.env` is git-ignored —
    never commit it.

2.  **Deploy**:

    ```bash
    docker compose up -d
    ```

3.  **Register User**:
    ```bash
    docker exec -it gorrson-server ./gorrson register --user <admin> --pass <password>
    ```

4.  **Open** [http://localhost:7077](http://localhost:7077) (or your `GORRSON_PORT_HOST`).

**Container hardening**: the image runs as the non-root user `gorrson` (uid 1000) under
`tini` as PID 1, the compose healthcheck polls `GET /health`, container memory is capped
at `512M`, and logs are rotated by Docker's `local` driver (10 MB × 3 files).

---

### Manual Installation

Ensure you have **Go 1.25.8+** installed (see the `go` directive in `go.mod`).

`make build` produces the **`rss-cli`** binary (Svelte UI embedded). The Docker image
builds the very same binary under the name **`gorrson`** — only the file name differs, and
every command below works with either one.

1.  **Build**: `make build`
2.  **Register**: `./rss-cli register --user <name> --pass <secret>`
3.  **Serve**: `GORRSON_JWT_SECRET=$(openssl rand -base64 32) ./rss-cli serve`

> Outside Docker, `GORRSON_JWT_SECRET` is strongly recommended: if it is unset, the server
> generates a random secret on every start, so **all sessions are invalidated on restart**.
> Known public values (`change-me-in-production`, `secret`, `changeme`, ...) or secrets
> shorter than 32 characters make `serve` refuse to start.

---

## Configuration

| Variable              | Default                          | Description                                                                                                                                                                                                                                                                                                                                        |
| :-------------------- | :------------------------------- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GORRSON_JWT_SECRET`  | — (required in Docker)           | JWT signing secret, minimum 32 characters. `docker compose` fails closed if it is not defined in `.env`; the server refuses known public values and secrets shorter than 32 chars. If left unset outside Docker, a random per-run secret is generated (sessions do not survive restarts). Generate with `openssl rand -base64 32`.                   |
| `GORRSON_PORT`        | `7077`                           | HTTP listen port read by `serve` (`--port` overrides it). The Docker image and compose pin it to `7077`.                                                                                                                                                                                                                                            |
| `GORRSON_PORT_HOST`   | `7077`                           | Host port published by docker-compose (`$GORRSON_PORT_HOST:7077`). Inside the container the port is always `7077`.                                                                                                                                                                                                                                   |
| `GORRSON_DATA_DIR`    | `/app/data` (Docker)             | Data directory declared in the image/compose. The server itself resolves the directory through `serve --data-dir` (default `./data`, which lands on `/app/data` in the container because the working directory is `/app`).                                                                                                                            |
| `GOGC`, `GOMEMLIMIT`  | `50`, `20MiB` (Docker)           | Go runtime memory knobs baked into the image for low-RAM hosts.                                                                                                                                                                                                                                                                                    |

Relevant flags: `--data-dir/-d` (persistent, all commands), and for `serve`:
`--port/-p`, `--log-format/-f` (`text` or `json`), `--log-level/-l` (`debug`, `info`,
`warn`, `error`) and `--silent/-s`.

## Mobile Integration (Fever)

Gorrson is fully compatible with the Fever protocol. Connect your favorite mobile app using:

- **Server URL**: `http://your-server:7077/api/fever`
- **Auth**: Use your Gorrson username and password. The system handles the standard Fever MD5 authentication automatically.

## API Documentation

Gorrson provides a powerful REST API for all its features. Once the server is running, you can access:

- **Interactive Swagger UI**: [http://localhost:7077/docs](http://localhost:7077/docs) (Browse & test endpoints).
- **OpenAPI Spec**: [docs/openapi.yaml](docs/openapi.yaml) (Standard technical definition).
- **Quick Reference**: See [DOCUMENTATION.md](DOCUMENTATION.md) for a human-friendly API summary.

### Endpoint Overview

Authenticated endpoints accept a JWT via the `gorrson_auth` cookie, an
`Authorization: Bearer` header, a `?token=` query parameter (iframes/proxies), or an
`X-API-Key: user:key` header (programmatic API keys).

- **Authentication**: `POST /login`, `POST /refresh`, `POST /logout`.
- **News**: `GET /news` (filters: `unread`, `sort`, FTS5 `search`, `category`, `feed_id`,
  `limit`/`offset`), `GET /news/{id}/content` (on-demand full-content extraction),
  `PUT /news/read-all` (scoped by `category`, `feed_id`, `search`, `limit`),
  `PUT /news/{id}/read`, `PUT /news/{id}/favorite` (pass `?favorite=false` to clear).
- **Feeds**: `GET`/`POST /feeds` (add with `{"url", "category"}`), `POST /feeds/import`
  and `GET /feeds/export` (OPML), `PATCH /feeds/{id}` (title/category), `DELETE /feeds/{id}`.
- **Image & page proxy**: `GET /proxy?url=...` (disk-cached image proxy with content-type
  validation), `GET /proxy/page?url=...&feed_id=...` (sanitized, script-free reader view
  of a remote page).
- **Scraping recipes**: `POST /recipes/save` (store a per-domain extraction recipe),
  `POST /recipes/update` (refresh the recipe list in the background).
- **Categories**: `PUT /categories/{name}` (rename a category across all of its feeds).
- **Filters & config**: `GET`/`POST /filters`, `DELETE /filters/{id}`, plus
  `GET /config/export` and `POST /config/import` (filters as portable JSON).
- **API keys**: `GET`/`POST /api-keys`, `DELETE /api-keys/{id}`.
- **Stats**: `GET /stats`.
- **Backup**: `GET /backup/export`, `POST /backup/import` (see below).
- **System**: `GET /health` (unauthenticated, used by the Docker healthcheck),
  `GET /docs` (Swagger UI), `GET /openapi.yaml`, and the SPA fallback that serves the
  embedded Svelte UI.
- **Fever**: `GET`/`POST /api/fever` (no JWT; authenticates with the Fever `api_key`).

## Backup & Restore

Every user is a single self-contained SQLite file in the data directory, so backups are
per-user:

- **Export**: `GET /backup/export` downloads the user's database as
  `gorrson-<user>-backup-<date>.db`.
- **Restore**: `POST /backup/import` with the multipart field `file` (capped at 100 MB).
  The upload is verified to be a valid SQLite database, swapped in atomically, and the
  cached connection is dropped so the next request already reads the restored data.
- **Portable subsets**: `GET /config/export` / `POST /config/import` (filters as JSON)
  and `GET /feeds/export` / `POST /feeds/import` (subscriptions as OPML).

```bash
# Export
curl -H "Authorization: Bearer $TOKEN" http://localhost:7077/backup/export -o gorrson-backup.db

# Restore
curl -H "Authorization: Bearer $TOKEN" -F "file=@gorrson-backup.db" http://localhost:7077/backup/import
```

There is no `backup` CLI command: backup and restore go through the API (or copy the
`.db` files from the data directory while the server is stopped).

## Architecture & CLI

| Command    | Description                                        |
| :--------- | :------------------------------------------------- |
| `register` | Creates a new user, calculates Fever keys.         |
| `user`     | Account management (list, passwd, delete).         |
| `add`      | Discovers and adds a feed from any URL.            |
| `filter`   | Manage advanced filtering rules via CLI.           |
| `import`   | Bulk import feeds from OPML files.                 |
| `serve`    | Starts the API, Web UI, and Sync Workers.          |
| `list`     | Built-in CLI news viewer for terminal power users. |

All commands accept `--data-dir/-d` (default `./data`). Usage examples:

```bash
./rss-cli register --user alice --pass secret123
./rss-cli add --url https://example.com/feed.xml --user alice
./rss-cli import --file feeds.opml --user alice
./rss-cli list --user alice --unread
./rss-cli filter add --user alice --pattern "crypto" --action BLOCK
./rss-cli user passwd --user alice --pass newsecret
./rss-cli serve --port 7077
```

## Development

| Target                       | Description                                                                        |
| :--------------------------- | :--------------------------------------------------------------------------------- |
| `make install`               | Install frontend (npm) and backend (Go) dependencies.                              |
| `make build`                 | Build the Svelte UI and embed it into the `rss-cli` binary.                        |
| `make run`                   | Build and start the server on `http://localhost:7077`.                             |
| `make dev`                   | Frontend (Vite) and backend in parallel, for development.                          |
| `make demo`                  | Seed a demo user + feeds/filters in `./data/demo` and serve it (`demo` / `DemoPass123!`). |
| `make test`                  | Build the UI, then run the Go test suite.                                          |
| `make bench` / `make bench-extensive` | DB benchmarks (`bench-extensive` runs multi-CPU with `-cpu=1,2,4`).      |
| `make perf`                  | Full performance gate: `perf-seed` → `bench-file` → `test-load` → `perf-report`.   |
| `make perf-gate`             | Fail on regressions against `perf/load-baseline.json` (see `perf/README.md`).      |
| `make clean`                 | Remove the binary, the UI build, and `./data`.                                     |

## Why Gorrson?

Inspired by minimalist readers like Miniflux, but optimized for speed and editorial aesthetics:

- **Zero External Dependencies**: No PostgreSQL or Redis required. Uses an optimized embedded SQLite engine.
- **High Customization**: Custom accent colors, light/dark modes, and multiple layout styles (Magazine, Compact, Minimal, Image-only).
- **Resource Efficient**: Extremely low CPU/RAM footprint, designed to run 24/7 as a background daemon.

---

**Developed with Go. Private by design. Fast by nature.**
