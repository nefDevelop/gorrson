# Gorrson: Comprehensive Documentation

## 🛠️ CLI Reference (rss-cli)

### Global Flags

Available for all commands:

- `-d, --data-dir` (string): Path where user databases are stored. (Default: `./data`)

---

### `register`

Creates a new user account and its dedicated SQLite database.

- **Flags**:
  - `-n, --user` (string): **Required.** Username.
  - `-p, --pass` (string): **Required.** Password.
- **Example**: `./rss-cli register --user alice --pass secret123`

---

### `add`

Automatically discovers and adds a single feed from a URL.

- **Flags**:
  - `-u, --url` (string): **Required.** Website or feed URL.
  - `-n, --user` (string): **Required.** Target user database.
- **Example**: `./rss-cli add --url https://news.ycombinator.com --user alice`

---

### `list`

Displays the latest 50 news entries stored in the database.

- **Flags**:
  - `-n, --user` (string): User database to query. (Default: `default`)
  - `-u, --unread` (bool): Show only unread articles.
- **Example**: `./rss-cli list --user alice --unread`

---

### `import`

Bulk imports feeds from an OPML file.

- **Flags**:
  - `-f, --file` (string): **Required.** Path to the .opml file.
  - `-n, --user` (string): **Required.** Target user database.
- **Example**: `./rss-cli import --file myfeeds.opml --user alice`

---

### `filter`

Manage content filtering rules (BLOCK/REPLACE).

- **Sub-commands**:
  - `add`: Create a new rule.
    - `-n, --user` (string): User identifier.
    - `-p, --pattern` (string): **Required.** Regex to match.
    - `-a, --action` (string): `BLOCK` or `REPLACE`. (Default: `BLOCK`)
    - `-t, --target` (string): `title`, `content`, or `both`. (Default: `both`)
    - `-f, --feed` (string): Feed ID (omit for Global filter).
    - `-r, --replacement` (string): Text to insert (for REPLACE action).
  - `list`: Show all filters for a user.
    - `-n, --user` (string): User identifier.
  - `remove`: Delete a filter.
    - `-n, --user` (string): User identifier.
    - `-i, --id` (string): **Required.** Filter ID.
- **Example**: `./rss-cli filter add -n alice -p "crypto" -a BLOCK`

---

### `user`

Advanced account management.

- **Sub-commands**:
  - `list`: List all users with a `.db` file.
  - `passwd`: Change a user's password.
    - `-u, --user` (string): **Required.** Username.
    - `-p, --pass` (string): **Required.** New password.
  - `delete`: Permanently delete a user and their data.
    - `-u, --user` (string): **Required.** Username.
- **Example**: `./rss-cli user passwd -u alice -p newsecret`

---

### `serve`

Starts the REST API and the background feed synchronization workers.

- **Flags**:
  - `-p, --port` (int): Port to listen on. (Default: `7077`)
  - `-f, --log-format` (string): `text` or `json`. (Default: `text`)
- **Example**: `./rss-cli serve --port 8080`

---

## 🌐 API Reference

Base URL: `http://localhost:7077` (Default)

### 🏥 System

| Method | Endpoint        | Description                             |
| :----- | :-------------- | :-------------------------------------- |
| `GET`  | `/health`       | Server status check.                    |
| `GET`  | `/docs`         | Interactive API Documentation.          |
| `GET`  | `/openapi.yaml` | Raw OpenAPI Specification.              |
| `GET`  | `/proxy`        | **Image Proxy**. Query param `?url=...` |

---

### 🔑 Authentication

| Method | Endpoint   | Payload                         | Description              |
| :----- | :--------- | :------------------------------ | :----------------------- |
| `POST` | `/login`   | `{"username", "password"}`      | Get JWT & Refresh token. |
| `POST` | `/refresh` | `{"username", "refresh_token"}` | Get new Access Token.    |
| `POST` | `/logout`  | `{"username", "refresh_token"}` | Invalidate session.      |

---

### 📰 News (Auth Required)

**Endpoint**: `/news`

- **GET**: List articles.
  - `unread=true`: Filter by unread status.
  - `sort=newest\|oldest\|favorite`: Order results.
  - `search=term`: Full-text search (FTS5).
  - `category=name`: Filter by feed category.
  - `limit=N` / `offset=N`: Pagination.
- **GET** `/news/{id}/content`: Fetches full article content (Reader Mode).
- **PUT** `/news/{id}/read`: Mark as read.
- **PUT** `/news/{id}/favorite`: Toggle favorite. Query param `?favorite=true\|false`.

---

### 📡 Feeds (Auth Required)

**Endpoint**: `/feeds`

- **GET**: List all subscriptions.
- **POST**: Add feed. Body: `{"url": "https://..."}`.
- **POST** `/feeds/import`: Upload OPML (Multipart or raw body).
- **GET** `/feeds/export`: Download OPML.
- **PATCH** `/feeds/{id}`: Update `title` or `category`.
- **DELETE** `/feeds/{id}`: Delete feed and all its articles.

---

### 🛡️ Filters (Auth Required)

**Endpoint**: `/filters`

- **GET**: List user filters.
- **POST**: Create filter. Body: `{"pattern", "action", "target", "replacement", "feed_id"}`.
- `DELETE` `/filters/{id}`: Remove a filter rule.
- `GET` `/config/export`: Export all filters as JSON.
- `POST` `/config/import`: Import filters from JSON.

---

### 💾 Data & Backup (Auth Required)

| Method | Endpoint          | Description                                    |
| :----- | :---------------- | :--------------------------------------------- |
| `GET`  | `/backup/export`  | Download a full SQLite database backup (.db).  |
| `POST` | `/backup/import`  | Restore entire database from a .db file.       |
| `GET`  | `/feeds/export`   | Download your subscriptions in OPML format.    |
| `POST` | `/feeds/import`   | Bulk import feeds from an OPML file.           |
| `GET`  | `/config/export`  | Export filters and settings as JSON.           |
| `POST` | `/config/import`  | Import settings and filters from JSON.         |

---

### 🔥 Fever API (Mobile Support)

**Endpoint**: `/api/fever`

- **Method**: `GET` / `POST`
- **Auth**: Query param `?api_key=...` (MD5 of `user:pass`).
- **Compatibility**: Supports groups, feeds, items, favicons, and unread marks.
- **Example**: `http://localhost:7077/api/fever?api_key=5927d3...`

---

**Gorrson: The Expert RSS Engine.**
