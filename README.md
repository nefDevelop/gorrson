# gorrson: High-Performance RSS CLI & API Engine

**gorrson** is a RSS engine written in Go. Robust, private, and multi-user RSS solution that runs anywhere.

## Key Features

1.  **Feed Discovery**: Automatically finds RSS/Atom feeds from any website URL.
2.  **Advanced Filtering**: Rule-based engine with Regex support. Actions: `BLOCK`, `HIDE`, `REPLACE`, `SILENT`, `HIGHLIGHT`. Targets: `title`, `content`, or `both`.
3.  **Full Content Scraping**: Intelligent reader mode extracts main text via `go-readability`. Robust charset detection (UTF-8, Latin1, CP1252, etc.).
4.  **Image Proxy & Cache**: Proxies images to protect privacy and caches them on disk.
5.  **SQLite FTS5**: Ultra-fast full-text search across all saved articles.
6.  **Fever API Support**: Compatible with mobile readers (Reeder, Unread, etc.).
7.  **Multi-Tenant**: Each user has a dedicated `.db` file for maximum privacy.

## Performance Benchmarks

Measured on AMD Ryzen 5 5600X with `-benchtime=5s`:

| Operation | CPU Cores | Latency (ns/op) | Throughput |
| :--- | :--- | :--- | :--- |
| **Full-Text Search** | 1 CPU | 4,526,220 | ~221 ops/sec |
| **Full-Text Search** | 2 CPU | 4,550,178 | ~220 ops/sec |
| **Full-Text Search** | 4 CPU | 4,568,430 | ~218 ops/sec |
| **List News (Feed)** | 1 CPU | 186,619 | ~5,358 ops/sec |
| **List News (Feed)** | 2 CPU | 177,149 | ~5,644 ops/sec |
| **List News (Feed)** | 4 CPU | 190,217 | ~5,257 ops/sec |

*Notice: Performance remains extremely stable across different CPU configurations. This confirms that Gorrson's efficiency is decoupled from high core counts, making it perfect for low-resource environments.*

## 🚀 Installation & Setup

### 🐳 Docker (Recommended for Servers)
The most robust way to deploy Gorrson is using Docker Compose.

1.  **Clone the repository**:
    ```bash
    git clone https://github.com/nef734/gorrson.git
    cd gorrson
    ```

2.  **Configure environment**:
    Copy the template and set a secure secret:
    ```bash
    cp .env.example .env
    # We recommend changing GORRSON_JWT_SECRET in the .env file
    ```

3.  **Deploy**:
    ```bash
    docker compose up -d
    ```

4.  **Create your first user**:
    Run the CLI within the container to register:
    ```bash
    docker exec -it gorrson-server ./gorrson register --user <admin> --pass <password>
    ```

---

### 🛠️ Manual Installation (Bare Metal)
Ensure you have **Go 1.21+** installed on your system.

1.  **Build the project**:
    ```bash
    make build
    ```

2.  **Initialize a user**:
    ```bash
    ./rss-cli register --user <name> --pass <secret>
    ```

3.  **Start the engine**:
    ```bash
    ./rss-cli serve
    ```

---

## 📖 Usage Guide

Once the server is running, you can interact with Gorrson via the API or the CLI.

### API Documentation
Gorrson includes interactive documentation:
- **Swagger UI**: [http://localhost:7077/docs](http://localhost:7077/docs)
- **OpenAPI Spec**: [http://localhost:7077/openapi.yaml](http://localhost:7077/openapi.yaml)

### Mobile Integration (Fever)
To use Gorrson with mobile apps like Reeder or Unread:
1. Use the **Fever API** endpoint: `http://your-server:7077/api/fever`
2. Authenticate with your username and password.


## Architecture & CLI

| Command | Description |
| :--- | :--- |
| `register` | Creates a new user and database. |
| `user` | Management (list, passwd, delete). |
| `add` | Discovers and adds a feed. |
| `filter` | Manage rules (`BLOCK`, `REPLACE`, etc.). |
| `import` | Bulk import feeds from OPML files. |
| `serve` | Starts the API and Sync Workers. |
| `list` | CLI news viewer. |

## Why Gorrson?
Inspired by minimalist readers like Miniflux, but optimized for portability:
- **Zero-Dependency**: No PostgreSQL required; uses embedded SQLite (CGO-free).
- **Privacy**: Each user owns their physical database file.
- **Resource Efficient**: Low CPU/RAM footprint, ideal for background daemons.

---
**Developed with Go.**
