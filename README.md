# 🚀 gorrson: High-Performance RSS CLI & API Engine

**gorrson** is a professional-grade RSS engine written in Go. Designed for developers and power users who need a robust, private, and multi-user RSS solution that runs anywhere—from a cloud server to a mobile device (Termux).

## ✨ Key Features (Point-by-Point)

1.  **🔍 Feed Discovery**: Automatically finds RSS/Atom feeds from any website URL. No need to hunt for the XML link.
2.  **🛡️ Advanced Filtering**: Rule-based engine with Regex support to `BLOCK` or `REPLACE` content (spam, ads, or reformatting).
3.  **📖 Full Content Scraping**: Intelligent reader mode extracts the main article text when the RSS only provides a summary. Includes automatic charset detection (UTF-8/Latin1).
4.  **🕵️ Image Proxy & Cache**: Protects your privacy by proxying images through the server. Features a local disk cache to save data and battery.
5.  **⚡ SQLite FTS5**: Ultra-fast full-text search across all your saved articles using SQLite's native virtual tables.
6.  **🔐 Security First**: JWT-based authentication, Argon2id password hashing, and `bluemonday` HTML sanitization.
7.  **📦 Multi-Tenant**: Each user has a dedicated physical `.db` file for maximum privacy and portability.

## 🛠️ Installation & Setup

### ⚡ Quick Install (Linux/Termux)
Run the automated installer to build the binary and setup directories:
```bash
chmod +x install.sh && ./install.sh
```

### 🐳 Docker (Recommended for Servers)
Deploy instantly with Docker Compose:
```bash
docker-compose up -d
```

### 🔨 Manual Build
```bash
make build
./rss-cli --help
```

## 🏗️ Architecture & CLI

| Command | Description |
| :--- | :--- |
| `register` | Creates a new user and its dedicated database file. |
| `user` | Advanced account management (list, passwd, delete). |
| `add` | Automatically discovers and adds a feed from any URL. |
| `filter` | Manage Regex rules (`BLOCK`, `REPLACE`) for your feeds. |
| `serve` | Starts the REST API and the background Sync Workers. |
| `list` | Quick CLI view of the latest news repository. |

### 🌐 REST API Highlights
- **GET `/news`**: List articles with `unread`, `sort`, `limit`, and `search` filters.
- **GET `/news/{id}/content`**: Smart full-content scraping with charset detection.
- **PUT `/categories/{name}`**: Bulk rename categories across multiple feeds.
- **GET/POST `/filters`**: Programmable Regex-based content filtering.
- **GET `/proxy?url=...`**: Securely fetch and cache images.
- **POST `/feeds/import`**: Bulk import feeds from OPML files.
- **GET `/docs`**: Interactive API documentation (Rapidoc).

## 👷 CI/CD & Releases
Gorrson is built for professional workflows:
- **GitHub Actions**: Automated testing and multi-platform releases (Linux, macOS, Windows).
- **Gitea Actions**: Fully compatible with self-hosted Gitea runners.
- **Automated Releases**: Binaries for `amd64` and `arm64` (Raspberry Pi/Mobile) are generated on every tag.

## ⚖️ Why Gorrson?
Inspired by minimalist readers like Miniflux, but optimized for portability:
- **Zero-Dependency**: No PostgreSQL required; uses embedded SQLite (CGO-free).
- **Privacy**: Each user owns their physical database file.
- **Resource Efficient**: Low CPU/RAM footprint, ideal for background daemons.

---
**Developed with Go excellence.**
