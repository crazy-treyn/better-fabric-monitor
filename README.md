# README

## About

This is the official Wails Svelte template.

## Live Development

To run in live development mode, run `wails dev` in the project directory. This will run a Vite development
server that will provide very fast hot reload of your frontend changes. If you want to develop in a browser
and have access to your Go methods, there is also a dev server that runs on http://localhost:34115. Connect
to this in your browser, and you can call your Go code from devtools.

## Building

To build a redistributable, production mode package, use `wails build`.

## DuckDB Go Development Setup

This project uses [DuckDB Go v2](https://github.com/duckdb/duckdb-go) for local analytics and caching.

### Prerequisites
- Go 1.20 or newer
- No native DuckDB install required (the Go driver bundles the engine)

### Install/Update Dependencies

```powershell
go mod tidy
```

### Troubleshooting
- If you see errors about `CURRENT_TIMESTAMP`, make sure you are using DuckDB Go v2.5.0+ and the code uses `now()` instead.
- If you see `no such column` or schema errors, delete your local DB file (usually at `%APPDATA%\better-fabric-monitor\fabric_monitor.db` on Windows) and restart the app to recreate it.
- If you see `dll not found` or `libduckdb` errors, ensure you are using the official Go driver and not the old `marcboeker/go-duckdb` package.

### Useful Links
- [DuckDB Go v2 Docs](https://github.com/duckdb/duckdb-go)
- [DuckDB SQL Reference](https://duckdb.org/docs/sql/introduction.html)
