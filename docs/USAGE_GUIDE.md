# 📖 Gorrson: Advanced Usage Guide

This guide provides practical examples for using Gorrson's professional features via CLI and API.

## 1. 🛡️ Advanced Filtering (Regex)

Gorrson allows you to clean your feeds using Regular Expressions. Rules can be **Global** (apply to all feeds) or **Specific** to a single feed.

### Example: Block "Sponsored" content (CLI)
```bash
./rss-cli filter add --user myuser --pattern "(?i)sponsored|promoted" --action BLOCK --target title
```

### Example: Rewrite "Go" to "Golang" (API)
**POST `/filters`**
```json
{
  "pattern": "\\bGo\\b",
  "action": "REPLACE",
  "target": "title",
  "replacement": "Golang"
}
```

## 2. 📖 Full Content Scraping

When a feed only provides a short summary, Gorrson can extract the full article text.

- **Automatic**: Triggered when you fetch an article's content via the API if it's not already stored.
- **Manual (CLI)**: Not needed, as the `serve` command handles background syncing and extraction.

**API Endpoint**: `GET /news/{id}/content`
- If the full content is already in the DB, it returns it instantly.
- If not, Gorrson's engine scrapes the original URL, cleans the noise, and saves the result.

## 3. 🕵️ Image Proxy & Privacy

To protect your IP address, use the built-in image proxy. It caches images on the server for 1 year.

**Usage Pattern**:
`http://your-server:7077/proxy?url=https://external-site.com/image.jpg`

**Features**:
- **Disk Cache**: Located in `data/cache/images/`.
- **Privacy**: The external server only sees the IP of your Gorrson instance.
- **Safety**: Limited to 5MB per image to prevent OOM attacks.

## 4. 🚀 API Quick Start (cURL)

### Login and get Token
```bash
curl -X POST http://localhost:7077/login \
     -H "Content-Type: application/json" \
     -d '{"username": "admin", "password": "password123"}'
```

### List Unread News (with Search)
```bash
curl -X GET "http://localhost:7077/news?unread=true&search=bitcoin" \
     -H "Authorization: Bearer <YOUR_TOKEN>"
```

### Add a new Feed (with Auto-Discovery)
```bash
curl -X POST http://localhost:7077/feeds \
     -H "Authorization: Bearer <YOUR_TOKEN>" \
     -H "Content-Type: application/json" \
     -d '{"url": "https://blog.golang.org"}'
```

## 5. 🐳 Docker Tips

### Viewing Logs
```bash
docker logs -f gorrson-server
```

### Accessing the DB inside the container
The databases are stored in the `./data` volume on your host. You can open them with any SQLite browser:
```bash
sqlite3 data/admin.db "SELECT title FROM entries LIMIT 5;"
```

---
**Gorrson: The Expert RSS Engine.**
