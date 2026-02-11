# Development

## Repository structure

```
cupertino/
  cli/       Go CLI
  web/       Next.js registry webapp
  docs/      Developer documentation
  install.sh Bootstrap installer
```

## CLI

Built with Go 1.24. Only external dependency is `go-sqlite3` for the local package database.

```bash
cd cli
go build -o cupertino .
```

The CLI stores installed packages under `/opt/cupertino/packages/<name>/<version>/` and symlinks binaries into `/opt/cupertino/bin/`. Package metadata is tracked in a local SQLite database at `/opt/cupertino/packages.db`.

The default registry URL is `http://localhost:8080` and can be overridden with `CUPERTINO_REGISTRY`.

## Web / Registry

Next.js app in `web/`. See [registry.md](registry.md) for API docs, deployment, and publishing.

```bash
cd web
cp .env.example .env.local
yarn install
yarn dev
```

### Environment variables

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | Neon Postgres connection string |
| `BLOB_READ_WRITE_TOKEN` | Vercel Blob access token |
| `ADMIN_API_KEY` | Secret key for admin API endpoints |
| `BASE_URL` | Public URL for constructing download links |
