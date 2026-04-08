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

## Installation & Setup

### Quick Install (Linux)
Clone the repository and run the automated installer:
```bash
git clone https://github.com/nef734/gorrson.git
cd gorrson
chmod +x install.sh && ./install.sh
```

### Docker (Recommended for Servers)
Deploy instantly with Docker Compose:
```bash
docker-compose up -d
```

### Manual Build
Ensure you have Go 1.21+ installed:
```bash
make build
./rss-cli --help
```

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
