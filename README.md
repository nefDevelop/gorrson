# 🚀 gorrson: High-Performance RSS CLI & API Engine

**gorrson** is a professional-grade RSS feed manager written in Go. It is designed for developers and power users who need a robust, portable, and multi-user RSS solution.

## 🏗️ Architecture Features

- **Multi-Tenant SQLite**: Each user gets a dedicated `.db` file in the `/data` directory (CGO-free via `modernc.org/sqlite`).
- **Sync Engine**: Concurrent background workers using Goroutines and Context for periodic feed updates.
- **Full-Text Search (FTS5)**: Ultra-fast article searching directly via SQLite Virtual Tables.
- **REST API**: Built with `go-chi`, featuring JWT authentication and monitoring.
- **Structured Logging**: Modern logging via `log/slog` with support for Text and JSON formats.
- **Graceful Shutdown**: Safe termination handling (SIGINT/SIGTERM) to prevent database corruption.
- **Security**: Content sanitization using `bluemonday` to prevent tracking and XSS.

## 🛠️ CLI Commands

The binary `rss-cli` provides the following specialized commands:

| Command | Description |
| :--- | :--- |
| `register` | Creates a new user and its dedicated database file. |
| `add` | Manually adds and syncs a single RSS feed. |
| `import` | Mass import from an OPML file. |
| `serve` | Starts the REST API and the background Sync Workers. |
| `list` | Quick CLI view of the user's news repository. |

### 🚩 Global Flags
- `-d, --data-dir`: Directory to store databases (default: `./data`).

### 🚩 `serve` Specific Flags
- `-p, --port`: Port to listen on (default: `7077`).
- `-f, --log-format`: Choose between `text` or `json` (ideal for daemons).

## 🌐 REST API Endpoints

Once the server is running (`serve`), you can interact with it via HTTP:

### 🏥 Monitoring
- **GET `/health`**: Returns `200 OK` if the server is alive.

### 🔑 Authentication
- **POST `/login`**: Send `{"username": "...", "password": "..."}` to receive a JWT.

### 📰 News Management (Requires JWT Bearer Token)
- **GET `/news`**: List articles.
    - `?unread=true`: Only unread entries.
    - `?sort=oldest|newest`: Control publication order.
    - `?search=keyword`: Full-text search across all titles and content.
- **PUT `/news/{id}/read`**: Marks a specific article as read.
- **PUT `/news/{id}/favorite`**: Toggle favorite status.

### 📡 Feed Management (Requires JWT Bearer Token)
- **GET `/feeds`**: List all subscribed feeds.
- **POST `/feeds`**: Add a new feed. Body: `{"url": "https://..."}`.
- **PATCH `/feeds/{id}`**: Update title or category.
- **DELETE `/feeds/{id}`**: Remove a feed and all its articles.

## 🚀 Running as a Daemon

A helper script is provided to run **gorrson** in the background:

1. **Start the daemon**:
   ```bash
   ./start_daemon.sh
   ```
   This will build the project, kill any previous instances, and start a new one using `nohup` with JSON logging.

2. **Monitor logs**:
   ```bash
   tail -f data/gorrson.log
   ```

## 🧪 Development & Testing

- `go test ./...`: Runs all unit and integration tests (API & CLI).
- `make build`: Compiles the tool for production.

## ⚖️ Gorrson vs. Miniflux

While inspired by minimalist readers like Miniflux, **gorrson** offers:
- **Zero-Dependency**: No PostgreSQL required; uses embedded SQLite.
- **Isolated Data**: Each user has their own physical `.db` file (Better privacy/portability).
- **CLI-First**: Designed to be driven from the terminal as much as from the API.
- **Structured JSON Logs**: Native support for modern observability stacks.

---

**Developed with Go excellence by Antigravity.**
