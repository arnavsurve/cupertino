# Registry

The registry is a Next.js app that serves as both a web UI for browsing packages and a REST API for the CLI.

## Stack

- **Runtime**: Next.js (App Router) on Vercel
- **Database**: Neon Postgres
- **File storage**: Vercel Blob (package tarballs)
- **Auth**: API key for admin endpoints

## Local development

```bash
cd web
cp .env.example .env.local
# Fill in DATABASE_URL, BLOB_READ_WRITE_TOKEN, ADMIN_API_KEY
yarn install
yarn dev
```

The app runs on `http://localhost:3000`.

## Deploy to Vercel

1. Push the repo to GitHub
2. Import the project in Vercel — set **Root Directory** to `web`
3. Add a Neon Postgres database (or link an existing one) — sets `DATABASE_URL`
4. Add Vercel Blob storage — sets `BLOB_READ_WRITE_TOKEN`
5. Set environment variables:
   - `ADMIN_API_KEY` — secret key for publishing packages
   - `BASE_URL` — your deployment URL (e.g. `https://cupertino.sh`)
6. Deploy

The database tables are created automatically on first query.

## API

All endpoints return JSON.

### Public

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/packages` | List packages (query: `limit`, `offset`) |
| `GET` | `/api/packages/:name` | Package info (all versions) |
| `GET` | `/api/packages/:name/:version` | Specific version details |
| `GET` | `/api/search?q=:query` | Search by name/description |
| `GET` | `/api/stats` | Registry statistics |
| `GET` | `/api/health` | Health check |
| `GET` | `/packages/:name-:version.tar.gz` | Download package tarball |

### Admin (requires `X-API-Key` or `Authorization: Bearer` header)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/packages` | Upload a package (multipart: `metadata` JSON + `file` tarball) |
| `PUT` | `/api/packages/:name` | Update metadata (description, homepage, license) |
| `DELETE` | `/api/packages/:name` | Delete all versions of a package |

## Publishing a package

```bash
# Create the tarball
tar -czf mypackage-1.0.0.tar.gz -C mypackage/ .

# Upload to the registry
curl -X POST https://cupertino.sh/api/packages \
  -H "X-API-Key: your-admin-key" \
  -F 'metadata={"name":"mypackage","version":"1.0.0","description":"My package","files":{"bin/mypackage":"bin/mypackage"}}' \
  -F "file=@mypackage-1.0.0.tar.gz"
```
