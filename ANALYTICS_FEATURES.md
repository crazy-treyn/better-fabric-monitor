# Analytics Features Implementation

## Summary
Two major features have been implemented:

### 1. Efficient Duration Formatting
- **Location**: `internal/utils/format.go`
- **Function**: `FormatDuration(durationMs int64) string`
- **Implementation**: Uses integer division only (no floating-point operations or memory allocations)
- **Format**:
  - Less than 60 seconds: `"45s"`
  - Less than 60 minutes: `"5m 30s"` or `"5m"`
  - 60 minutes or more: `"2h 15m"` or `"2h"`

**Performance**: Extremely fast - only uses integer division and modulo operations, no string allocations until final formatting.

### 2. Comprehensive Analytics Dashboard
A new analytics view that provides actionable insights for Fabric production maintainers.

#### New Database Queries (`internal/db/queries.go`)
- `GetDailyStats(days int)` - Job statistics grouped by day
- `GetWorkspaceStats(days int)` - Performance breakdown by workspace
- `GetItemTypeStats(days int)` - Job statistics by item type (Pipeline, Notebook, etc.)
- `GetRecentFailures(limit int)` - Most recent failed jobs with details
- `GetLongRunningJobs(days, minDeviationPct, limit)` - Jobs that took significantly longer than average

#### New Data Models (`internal/db/models.go`)
- `DailyStats` - Daily aggregated metrics
- `WorkspaceStats` - Workspace-level performance
- `ItemTypeStats` - Item type breakdown
- `RecentFailure` - Failed job details
- `LongRunningJob` - Anomaly detection for slow jobs

#### Backend API (`app.go`)
- `GetAnalytics(days int)` - Single endpoint that returns all analytics data
- Returns:
  - Overall statistics (total, success rate, average duration)
  - Daily trends
  - Workspace performance
  - Item type breakdown
  - Recent failures
  - Long-running job anomalies (50%+ above average)

#### Frontend Components
**New Component**: `frontend/src/components/Analytics.svelte`

**Features**:
1. **Time Period Selection** - 24 hours, 7/14/30/90 days
2. **Overall Stats Cards**:
   - Total Jobs
   - Successful Jobs (green)
   - Failed Jobs (red)
   - Success Rate %
   - Average Duration (formatted)

3. **Daily Trend View** - Shows daily performance with success/failure counts and rates

4. **Workspace Performance** - Compare workspaces by:
   - Total job count
   - Success/failure breakdown
   - Success rate
   - Average duration

5. **Job Type Breakdown** - Statistics by item type (Pipeline, Notebook, Dataflow, etc.)

6. **Recent Failures Panel** - Last 10 failures with:
   - Job name and workspace
   - Failure reason
   - Timestamp
   - Visual red-themed alert panel

7. **Long Running Jobs Table** - Detects anomalies:
   - Shows jobs running 50%+ longer than their historical average
   - Displays actual duration vs. average
   - Shows percentage deviation
   - Helps identify performance regressions

**Updated Component**: `frontend/src/components/Dashboard.svelte`
- Added navigation tabs between "Jobs" and "Analytics" views
- Integrated `formatDuration()` helper for consistent duration display
- Analytics tab available next to Jobs view

## Usage

### For End Users
1. Navigate to the application
2. Click the "Analytics" tab in the header
3. Select your desired time period (default: 7 days)
4. Review key metrics:
   - Check success rates across workspaces
   - Identify failing jobs in the "Recent Failures" section
   - Spot performance issues in "Long Running Jobs"
   - Monitor daily trends

### For Developers
```go
// Use the duration formatter
import "better-fabric-monitor/internal/utils"

formatted := utils.FormatDuration(125000) // "2m 5s"
formatted := utils.FormatDuration(3665000) // "1h 1m"
```

```go
// Get analytics data
analytics := app.GetAnalytics(7) // Last 7 days
```

## Performance Notes
- All analytics queries run against local DuckDB cache (no API calls)
- Queries use aggregations and joins for efficiency
- Frontend duration formatting matches backend for consistency
- Analytics view loads independently from jobs view

## Key Insights for Maintainers
1. **Daily Success Rate** - Quickly identify problem days
2. **Workspace Comparison** - See which workspaces have reliability issues
3. **Job Type Analysis** - Understand which types of jobs fail most often
4. **Failure Tracking** - Recent failures with reasons for quick triage
5. **Performance Anomalies** - Catch jobs that are running slower than usual (early warning for issues)

## Future Enhancements (Optional)
- Add charts/graphs for visual trends
- Export analytics to CSV/Excel
- Email/Slack alerts for failures
- SLA tracking and breach notifications
- Historical comparison (week-over-week, month-over-month)
