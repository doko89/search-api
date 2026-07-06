# Search API

Web search API dengan multi-provider backend (DuckDuckGo, Brave, Tavily, Firecrawl, Exa). Mendukung REST API dan MCP (Model Context Protocol) Streamable HTTP Transport.

## Search Providers

Provider dicoba secara berurutan sesuai `SEARCH_PROVIDER`. Jika satu gagal, fallback ke provider berikutnya.

| Provider    | Requires API Key | Web Search | Image Search |
|-------------|------------------|------------|--------------|
| DuckDuckGo  | No               | ✔ HTML scrape | ✔ Scrape |
| Brave       | Yes              | ✔ API      | ✔ API |
| Tavily      | Yes              | ✔ API      | ✔ API |
| Firecrawl   | Yes              | ✔ API      | ✘ |
| Exa         | Yes              | ✔ Neural API | ✘ |

## Endpoints

### REST API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/search?q=<query>&provider=<name>` | Cari di web |
| `GET` | `/images?q=<query>&provider=<name>` | Cari gambar (brave/tavily) |
| `GET` | `/health` | Health check |

Contoh:
```bash
# Tanpa provider (fallback otomatis)
curl -H "Authorization: Bearer <token>" "https://<domain>/search?q=golang"

# Pilih provider spesifik
curl -H "Authorization: Bearer <token>" "https://<domain>/search?q=golang&provider=brave"

# Cari gambar
curl -H "Authorization: Bearer <token>" "https://<domain>/images?q=kucing"
```

### MCP (Model Context Protocol)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/mcp` | JSON-RPC endpoint |

Tool yang tersedia:

- **`search_web`** — Cari web dengan multi-provider
- **`search_images`** — Cari gambar (brave/tavily)

Parameter:
- `query` (required) — kata kunci pencarian
- `provider` (optional) — pilih provider: `duckduckgo`, `brave`, `tavily`, `firecrawl`

Contoh:
```bash
# List tools
curl -X POST "https://<domain>/mcp" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'

# Panggil search dengan provider tertentu
curl -X POST "https://<domain>/mcp" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"search_web","arguments":{"query":"golang","provider":"tavily"}}}'

# Cari gambar
curl -X POST "https://<domain>/mcp" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search_images","arguments":{"query":"kucing"}}}'
```

### Konfigurasi MCP Client (opencode)

```json
{
  "mcpServers": {
    "search-api": {
      "url": "https://<domain>/mcp",
      "headers": {
        "Authorization": "Bearer dev-secret-token"
      }
    }
  }
}
```

## Autentikasi

Semua endpoint menggunakan Bearer token. Set `API_TOKEN` di `.env`:

```env
API_TOKEN=your-secret-token
BRAVE_API_KEY=your_brave_key
TAVILY_API_KEY=your_tavily_key
FIRECRAWL_API_KEY=your_firecrawl_key
EXA_API_KEY=your_exa_key
SEARCH_PROVIDER=duckduckgo,brave,tavily,firecrawl,exa
```

Default `API_TOKEN` di kode fallback ke `dev-secret-token` (hanya untuk development).

## Run Lokal

```bash
cp .env.example .env  # lalu isi API key
go run ./cmd/api/
```

## Build untuk ARM64

```bash
GOOS=linux GOARCH=arm64 go build -o search-api-linux-arm64 ./cmd/api/
```

## Deploy

### Systemd

Service file: `search-api.service`. Environment variables via `/apps/search-api/.env`.

```bash
scp search-api-linux-arm64 user@host:/apps/search-api/search-api
ssh user@host "sudo systemctl restart search-api"
```

### Caddy

```
https://<domain> {
    encode zstd gzip
    reverse_proxy :8080
}
```
