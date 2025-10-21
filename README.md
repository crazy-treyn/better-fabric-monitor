# Better Fabric Monitor

A desktop application for monitoring Microsoft Fabric job executions, built with Wails (Go + Svelte) and DuckDB for local analytics.

## Features

- 🔐 **Azure AD Authentication** - Secure device code flow authentication
- 📊 **Job Monitoring** - Track pipeline, notebook, and other Fabric job executions
- 💾 **Local Caching** - DuckDB-powered local database for offline access
- 📈 **Analytics Dashboard** - Success/failure rates, recent failures, and anomaly detection
- 🔄 **Incremental Sync** - Efficient API usage with smart caching
- ⚡ **Parallel API Processing** - Concurrent workspace and item fetching with adaptive rate limiting
- 🔁 **Intelligent Retry Logic** - Exponential backoff with automatic throttle detection
- 🎨 **Modern UI** - Dark-themed interface built with Tailwind CSS

## Prerequisites

- **Go 1.20+** - [Download](https://go.dev/dl/)
- **Node.js 16+** - [Download](https://nodejs.org/)
- **Wails CLI** - `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **WebView2 Runtime** - Pre-installed on Windows 10/11
- **GCC Compiler** - [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) or [MinGW-w64](https://www.mingw-w64.org/)

## Quick Start

1. **Clone and install dependencies**
   ```powershell
   git clone https://github.com/crazy-treyn/better-fabric-monitor.git
   cd better-fabric-monitor
   go mod tidy
   cd frontend; npm install; cd ..
   ```

2. **Run in development mode**
   ```powershell
   wails dev
   ```

3. **Build for production**
   ```powershell
   wails build
   .\build\bin\better-fabric-monitor.exe
   ```

## Usage

1. **Authenticate** - Enter your Azure Tenant ID and follow the device code flow
2. **Load Data** - Click "Refresh from API" to fetch workspaces and jobs (typically 6-10 seconds)
3. **View Analytics** - Switch to the Analytics tab for insights and metrics

### Performance & API Optimization

The application uses several techniques to maximize performance:

- **Parallel Processing**: Fetches data from multiple workspaces simultaneously (up to 8 concurrent)
- **Adaptive Rate Limiting**: Starts at 50 requests/second, dynamically adjusts based on API responses
- **Intelligent Retry**: Automatic retry with exponential backoff (1s → 2s → 4s → 8s → 16s → 32s)
- **429 Handling**: Respects `Retry-After` headers and reduces request rate when throttled
- **Incremental Sync**: Only fetches new jobs since last sync, avoiding redundant API calls

**Performance**: Typical refresh with 8 workspaces and 160 items completes in 6-10 seconds.

## Database

The application uses DuckDB to store workspace, item, and job instance data locally at `data/fabric-monitor.db`.

**Key Improvements:**
- All timestamps stored in **UTC** and converted to local time in the UI
- Analytics queries use `CURRENT_TIMESTAMP` for accurate time-based filtering (fixes "Last 24 Hours" display)

**Inspect with DuckDB CLI:**
```powershell
duckdb data\fabric-monitor.db
SELECT * FROM workspaces;
SELECT * FROM job_instances WHERE status = 'Failed' ORDER BY start_time DESC LIMIT 10;
```

**Configure location** (optional):
```powershell
$env:FABRIC_MONITOR_DATABASE_PATH = "C:\path\to\your\database.db"
```

**Logging:**
- API performance: `data/api_performance.log`
- API errors: `data/api_errors.log`  
- Throttle events: `data/api_throttle.log`

## Project Structure

```
better-fabric-monitor/
├── app.go                      # Main app logic and Wails bindings
├── main.go                     # Application entry point
├── internal/
│   ├── auth/                   # Azure AD authentication
│   ├── config/                 # Configuration management
│   ├── db/                     # DuckDB database layer
│   ├── fabric/                 # Microsoft Fabric API client
│   └── utils/                  # Utility functions
├── frontend/src/
│   ├── components/             # Svelte components
│   └── stores/                 # State management
├── data/                       # Database storage
└── build/                      # Build output
```

## Technology Stack

- **Backend**: Go 1.20+ with Wails v2
- **Frontend**: Svelte + Vite + Tailwind CSS
- **Database**: DuckDB (embedded analytical database)
- **Authentication**: Azure AD device code flow
- **API**: Microsoft Fabric REST API

## Resources

- [Wails Documentation](https://wails.io/docs/introduction)
- [DuckDB Documentation](https://duckdb.org/docs/)
- [Microsoft Fabric API](https://learn.microsoft.com/en-us/rest/api/fabric/)
- [Svelte Documentation](https://svelte.dev/docs)

## License

See LICENSE file for details.

