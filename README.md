# 🚀 gorrson: High-Performance RSS CLI & API Engine

**gorrson** is a professional-grade RSS feed manager written in Go. It is designed for developers and power users who need a robust, portable, and multi-user RSS solution.

## 🏗️ Architecture Features

- **Multi-Tenant SQLite**: Each user gets a dedicated `.db` file in the `/data` directory (CGO-free via `modernc.org/sqlite`).
- **Sync Engine**: Concurrent background workers using Goroutines and Context for periodic feed updates.
- **Full-Text Search (FTS5)**: Ultra-fast article searching directly via SQLite Virtual Tables.
- **REST API**: Built with `go-chi`, featuring JWT authentication.
- **Security**: Content sanitization using `bluemonday` to prevent tracking and XSS.
- **OPML Support**: Standardized import/export of feed lists.

## 🛠️ CLI Commands

The binary `rss-cli` provides the following specialized commands:

| Command | Description |
| :--- | :--- |
| `register --user [n] --pass [p]` | Creates a new user and its dedicated database file. |
| `add --url [URL] --user [n]` | Manually adds and syncs a single RSS feed. |
| `import --file feeds.opml --user [n]` | Mass import from an OPML file. |
| `serve --port 7077` | Starts the REST API and the background Sync Workers. |
| `list --user [n] [--unread]` | Quick CLI view of the user's news repository. |

## 🌐 REST API Endpoints

Once the server is running (`serve`), you can interact with it via HTTP:

### 🔑 Authentication
- **POST `/login`**: Send `{"username": "...", "password": "..."}` to receive a JWT.

### 📰 News Management (Requires JWT Bearer Token)
- **GET `/news`**: List articles.
    - `?unread=true`: Only unread entries.
    - `?sort=oldest|newest`: Control publication order.
    - `?search=keyword`: Full-text search across all titles and content.
- **PUT `/news/{id}/read`**: Marks a specific article as read.

### 📡 Feed Management (Requires JWT Bearer Token)
- **GET `/feeds`**: List all subscribed feeds.
- **POST `/feeds`**: Add a new feed. Body: `{"url": "https://..."}`.
- **PATCH `/feeds/{id}`**: Update title or category. Body: `{"title": "...", "category": "..."}`.
- **DELETE `/feeds/{id}`**: Remove a feed and all its articles.

## 🚀 Getting Started

### 📦 Build
```bash
make build
```

### 👤 Register a user
```bash
./rss-cli register --user mynode --pass strongpassword
```

### 📡 Import feeds
```bash
./rss-cli import --file my_feeds.opml --user mynode
```

### 🚦 Start the engine
```bash
./rss-cli serve --port 7077
```

## 🧪 Development & Testing

Use the provided **Makefile** for streamlined workflows:

- `make test`: Runs all database and parser integration tests.
- `make clean`: Resets the environment (removes databases and binaries).
- `make build`: Compiles the tool for production.

---

**Developed with Go excellence by Antigravity.**
