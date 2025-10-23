# Better Fabric Monitor

**Finally, the Microsoft Fabric monitoring experience you've been waiting for.**

Better Fabric Monitor is a desktop application that gives you the historical insights and analytics that Microsoft Fabric's built-in monitoring simply doesn't provide. Built for data professionals who need to understand job execution trends, identify patterns, and troubleshoot issues across their Fabric workspaces.

## Screenshots

<!-- TODO: Add screenshot of main dashboard view -->
![Dashboard View]()

<!-- TODO: Add screenshot of analytics dashboard -->
![Analytics Dashboard]()

## Why Better Fabric Monitor?

If you've struggled with Fabric's limited monitoring capabilities, you're not alone. Better Fabric Monitor solves these problems:

- **Historical Trends** - View job execution patterns over days and weeks, not just recent runs
- **Cross-Workspace Analytics** - Aggregate insights across all your workspaces in one place
- **Failure Analysis** - Quickly identify problematic jobs with detailed error messages and failure trends
- **Performance Insights** - Spot long-running jobs and performance degradation before they become critical
- **Offline Access** - Review historical data even when disconnected from the API

## Key Features

### 📊 Comprehensive Job Monitoring
- **Real-time job tracking** across all your Fabric workspaces (Pipelines, Notebooks, and more)
- **Hierarchical execution view** - Drill into pipeline executions to see child activities and nested pipeline calls
- **Advanced filtering** - Search by job name, filter by type, status, or workspace
- **Multi-workspace selection** - Monitor multiple workspaces simultaneously

### 📈 Powerful Analytics Dashboard
Get insights that Fabric doesn't give you out of the box:

- **Success/Failure Trends** - Interactive daily charts showing job outcomes over time
- **Workspace Performance** - Compare success rates and execution patterns across workspaces
- **Job Type Analysis** - Understand which item types are running (or failing) most frequently
- **Recent Failures** - Quick view of latest failures with full error details
- **Long-Running Job Detection** - Automatically identifies jobs taking significantly longer than average
- **Flexible Time Ranges** - Analyze 7, 14, 30, or 90 days of historical data

### 🔍 Interactive Drill-Down
Every chart is clickable - drill down from high-level metrics to item-specific details:

- Click a date to see all jobs that ran on that day
- Click a workspace to view performance by individual items
- Click a job type to analyze all items of that type
- View detailed statistics including execution counts, success rates, and average durations

### ⚡ Fast & Efficient
- **Lightning-fast sync** - Typical data refresh completes in seconds
- **Smart incremental updates** - Only fetches new data since your last sync
- **Local database** - Uses DuckDB for instant queries and offline access
- **Optimized API usage** - Intelligent parallel processing and rate limiting respects Fabric API limits

### 🔐 Secure Authentication
- Works with your **Entra ID** (Azure AD) credentials
- Simple device code authentication flow
- No passwords stored locally

### 💾 Your Data, Your Control
- All data stored locally in a DuckDB database at `data/fabric-monitor.db`
- Run your own custom SQL queries for advanced analysis
- Export capabilities for reporting

## Getting Started

### Prerequisites

For end users downloading a release:
- **Windows 10 or 11** (WebView2 Runtime pre-installed on modern Windows)
- **Microsoft Fabric Access** with appropriate permissions

For developers building from source:
- **Go 1.24+** - [Download](https://go.dev/dl/)
- **Node.js 24+** - [Download](https://nodejs.org/)
- **Wails CLI** - `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Installation

#### Option 1: Download Pre-built Release
1. Download the latest `better-fabric-monitor.exe` from [Releases](https://github.com/crazy-treyn/better-fabric-monitor/releases)
2. Run the executable - no installation required

#### Option 2: Build from Source
```powershell
# Clone the repository
git clone https://github.com/crazy-treyn/better-fabric-monitor.git
cd better-fabric-monitor

# Install dependencies
go mod tidy
cd frontend
npm install
cd ..

# Build the application
wails build

# Run the built executable
.\build\bin\better-fabric-monitor.exe
```

### First Run

1. **Launch the application**
2. **Enter your Azure Tenant ID** (your organization's Entra ID tenant)
3. **Follow the device code authentication** - Copy the code and authenticate in your browser
4. **Click "Refresh from API"** to load your workspaces and jobs
5. **Explore!** - Switch between Jobs and Analytics views

## Usage Guide

### Jobs View
- Browse all recent job executions across your workspaces
- Use filters to narrow down to specific items, types, or statuses
- Click the expand arrow (▶) next to pipeline jobs to see child activities
- Nested pipelines show their own child activities with multiple levels of hierarchy

### Analytics View
- View overall statistics for your selected time period
- **Click any chart element** to drill down into detailed item-level data
- Use workspace and job type filters to focus your analysis
- Search by item name to find specific items
- Review "Recent Failures" section for error details
- Check "Long-Running Jobs" to identify performance issues

### Advanced: Custom Queries
Power users can query the local DuckDB database directly:

```powershell
# Install DuckDB CLI
# Download from https://duckdb.org/docs/installation/

# Open the database
duckdb data\fabric-monitor.db

# Example queries
SELECT * FROM workspaces;
SELECT * FROM items WHERE type = 'DataPipeline';
SELECT * FROM job_instances WHERE status = 'Failed' ORDER BY start_time DESC LIMIT 10;
```

## Technical Architecture

### High-Level Design
- **Frontend**: Svelte with Tailwind CSS for a modern, responsive UI
- **Backend**: Go with Wails v2 for native desktop performance
- **Data Store**: DuckDB embedded database optimized for analytical queries
- **API Integration**: Intelligent polling with parallel processing and adaptive rate limiting

### Performance Strategy
The application is designed for speed:
- Parallel workspace and item fetching (up to 8 concurrent requests)
- Smart incremental sync - only fetches jobs since last update
- Local DuckDB caching eliminates redundant API calls
- All analytics calculations performed in DuckDB using SQL for optimal performance

### Data Management
- Database location: `data/fabric-monitor.db` (customizable via `FABRIC_MONITOR_DATABASE_PATH` environment variable)
- All timestamps stored in UTC, displayed in local time
- Automatic retry logic with exponential backoff handles API throttling

## Development

### Running in Development Mode
```powershell
wails dev
```

This starts a development server with hot reload for frontend changes.

### Project Structure
```
better-fabric-monitor/
├── app.go                      # Main app logic and Wails bindings
├── main.go                     # Application entry point
├── internal/
│   ├── auth/                   # Entra ID authentication
│   ├── config/                 # Configuration management
│   ├── db/                     # DuckDB database layer
│   ├── fabric/                 # Microsoft Fabric API client
│   └── utils/                  # Utility functions
├── frontend/src/
│   ├── components/             # Svelte UI components
│   └── stores/                 # State management
└── data/                       # Local database and logs
```

## Resources

- [Microsoft Fabric Documentation](https://learn.microsoft.com/en-us/fabric/)
- [Microsoft Fabric REST API](https://learn.microsoft.com/en-us/rest/api/fabric/)
- [Wails Framework](https://wails.io/docs/introduction)
- [DuckDB](https://duckdb.org/docs/)

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

See LICENSE file for details.

