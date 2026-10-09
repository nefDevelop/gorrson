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

1.  **Deploy**:

    ```bash
    cp .env.example .env
    # Rellena GORRSON_JWT_SECRET en .env (openssl rand -base64 32)
    docker compose up -d
    ```

2.  **Register User**:
    ```bash
    docker exec -it gorrson-server ./gorrson register --user <admin> --pass <password>
    ```

---

### Manual Installation

Ensure you have **Go 1.22+** installed.

1.  **Build**: `make build`
2.  **Register**: `./rss-cli register --user <name> --pass <secret>`
3.  **Serve**: `./rss-cli serve`

---

## Mobile Integration (Fever)

Gorrson is fully compatible with the Fever protocol. Connect your favorite mobile app using:

- **Server URL**: `http://your-server:7077/api/fever`
- **Auth**: Use your Gorrson username and password. The system handles the standard Fever MD5 authentication automatically.

## API Documentation

Gorrson provides a powerful REST API for all its features. Once the server is running, you can access:

- **Interactive Swagger UI**: [http://localhost:7077/docs](http://localhost:7077/docs) (Browse & test endpoints).
- **OpenAPI Spec**: [docs/openapi.yaml](docs/openapi.yaml) (Standard technical definition).
- **Quick Reference**: See [DOCUMENTATION.md](DOCUMENTATION.md) for a human-friendly API summary.

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

## Why Gorrson?

Inspired by minimalist readers like Miniflux, but optimized for speed and editorial aesthetics:

- **Zero External Dependencies**: No PostgreSQL or Redis required. Uses an optimized embedded SQLite engine.
- **High Customization**: Custom accent colors, light/dark modes, and multiple layout styles (Magazine, Compact, Minimal, Image-only).
- **Resource Efficient**: Extremely low CPU/RAM footprint, designed to run 24/7 as a background daemon.

---

**Developed with Go. Private by design. Fast by nature.**
