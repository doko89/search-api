# Search API

Web search API dengan multi-provider backend (DuckDuckGo, Brave, Tavily, Firecrawl). Mendukung REST API dan MCP (Model Context Protocol) Streamable HTTP Transport.

## Search Providers

Provider dicoba secara berurutan sesuai `SEARCH_PROVIDER`. Jika satu gagal, fallback ke provider berikutnya.

| Provider    | Requires API Key | Type      |
|-------------|------------------|-----------|
| DuckDuckGo  | No               | HTML scrape |
| Brave       | Yes              | API       |
| Tavily      | Yes              | API       |
| Firecrawl   | Yes              | API       |

## Endpoints

### REST API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/search?q=<query>&provider=<name>` | Cari di web |
| `GET` | `/health` | Health check |

Contoh:
```bash
# Tanpa provider (fallback otomatis)
curl -H "Authorization: Bearer <token>" "https://search.nex.my.id/search?q=golang"

# Pilih provider spesifik
curl -H "Authorization: Bearer <token>" "https://search.nex.my.id/search?q=golang&provider=brave"
```

### MCP (Model Context Protocol)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/mcp` | JSON-RPC endpoint |

Tool yang tersedia:

- **`search_web`** — Cari web dengan multi-provider

Parameter:
- `query` (required) — kata kunci pencarian
- `provider` (optional) — pilih provider: `duckduckgo`, `brave`, `tavily`, `firecrawl`

Contoh:
```bash
# List tools
curl -X POST "https://search.nex.my.id/mcp" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'

# Panggil search dengan provider tertentu
curl -X POST "https://search.nex.my.id/mcp" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"search_web","arguments":{"query":"golang","provider":"tavily"}}}'
```

### Konfigurasi MCP Client (opencode)

```json
{
  "mcpServers": {
    "search-api": {
      "url": "https://search.nex.my.id/mcp",
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
SEARCH_PROVIDER=duckduckgo,brave,tavily,firecrawl
```

Default: `API_TOKEN=dev-secret-token` (jika tidak diset).

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
https://search.nex.my.id {
    encode zstd gzip
    reverse_proxy :8080
}
```
