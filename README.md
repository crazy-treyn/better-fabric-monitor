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

## Database Location

The application stores data in a DuckDB database file located at:
- **Default path**: `data/fabric-monitor.db` (relative to the application's working directory)
- When running `wails dev`: Located in the project root directory at `data/fabric-monitor.db`
- When running the built .exe: Located relative to where the executable runs

### Finding Your Database

Check the console output when the app starts - it will print:
```
Initializing DuckDB database at: <absolute-path>
```

**Important**: If no database path is configured, DuckDB defaults to an **in-memory database** that is lost when the app closes! The app now validates this and ensures a file-based database is always used.

### Inspecting the Database

You can manually inspect the database using:
1. **DuckDB CLI**: Download from [duckdb.org](https://duckdb.org/docs/installation/)
   ```powershell
   duckdb data\fabric-monitor.db
   ```

The database contains:
- `workspaces` - Fabric workspace information
- `items` - Pipelines, notebooks, and other Fabric items  
- `job_instances` - Job execution history
- `pipeline_runs` - Pipeline run details
- `sync_metadata` - Synchronization tracking

### Configuring the Database Location

To change the database location, set the environment variable:
```powershell
$env:FABRIC_MONITOR_DATABASE_PATH = "C:\path\to\your\database.db"
```

Or create a `.env` file in the project root:
```
FABRIC_MONITOR_DATABASE_PATH=C:\path\to\your\database.db
```

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
- If you see `no such column` or schema errors, delete your local DB file (check console output for the exact path, default: `data\fabric-monitor.db`) and restart the app to recreate it.
- If you see `dll not found` or `libduckdb` errors, ensure you are using the official Go driver and not the old `marcboeker/go-duckdb` package.
- If data doesn't persist between runs, check that you're not using an in-memory database - the console should show "Initializing DuckDB database at:" with a file path (not empty).

### Useful Links
- [DuckDB Go v2 Docs](https://github.com/duckdb/duckdb-go)
- [DuckDB SQL Reference](https://duckdb.org/docs/sql/introduction.html)
