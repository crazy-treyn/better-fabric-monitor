# Better Fabric Monitor - Implementation Plan & Architecture

## Executive Summary

A native desktop application for monitoring Microsoft Fabric workspaces, pipelines, notebooks, and dataflows. Built with Wails (Go + Svelte), using DuckDB for local storage and native Entra ID authentication.

**Key Metrics:**
- Development Time: 3-4 weeks
- Binary Size: ~15MB
- Platforms: Windows, macOS, Linux
- Authentication: Entra ID (OAuth 2.0 / MSAL)
- Data Storage: DuckDB (embedded, local-first)

---

## Table of Contents

1. [Project Goals](#project-goals)
2. [Architecture Overview](#architecture-overview)
3. [Technology Stack](#technology-stack)
4. [System Architecture](#system-architecture)
5. [Component Specifications](#component-specifications)
6. [Database Schema](#database-schema)
7. [API Integration](#api-integration)
8. [Authentication Flow](#authentication-flow)
9. [File Structure](#file-structure)
10. [Implementation Phases](#implementation-phases)
11. [Development Setup](#development-setup)
12. [UI/UX Specifications](#uiux-specifications)
13. [Testing Strategy](#testing-strategy)
14. [Deployment Strategy](#deployment-strategy)

---

## Project Goals

### Primary Objectives
1. **Real-time Monitoring**: Track Fabric job runs, pipelines, notebooks, and dataflows
2. **Historical Analysis**: Store and query historical job execution data locally
3. **Smart Alerting**: Desktop notifications for failed jobs or long-running tasks
4. **Offline-First**: Work without constant internet connection, sync when available
5. **Zero Infrastructure**: No servers, no hosting, no database management

### Success Criteria
- ✅ Authenticate with Entra ID in < 30 seconds
- ✅ Sync 1000+ job runs in < 5 seconds
- ✅ Query historical data in < 100ms
- ✅ Binary size < 20MB
- ✅ Memory usage < 150MB idle
- ✅ Cross-platform compatibility (Windows, macOS, Linux)

---

## Architecture Overview

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Fabric Monitor Desktop App                │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              Svelte Frontend (UI Layer)               │  │
│  │  - Dashboard Views                                    │  │
│  │  - Gantt Charts (vis-timeline)                        │  │
│  │  - Filtering & Search                                 │  │
│  │  - Settings Management                                │  │
│  └───────────────────────────────────────────────────────┘  │
│                            ▲                                 │
│                            │ Wails Bridge                    │
│                            ▼                                 │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              Go Backend (Business Logic)              │  │
│  │                                                        │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌────────────┐  │  │
│  │  │ Auth Manager │  │ API Client   │  │  Poller    │  │  │
│  │  │   (MSAL)     │  │  (Fabric)    │  │ (Background)│  │  │
│  │  └──────────────┘  └──────────────┘  └────────────┘  │  │
│  │                                                        │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌────────────┐  │  │
│  │  │   DuckDB     │  │ Notification │  │   Config   │  │  │
│  │  │   Manager    │  │   Service    │  │  Manager   │  │  │
│  │  └──────────────┘  └──────────────┘  └────────────┘  │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                               │
└─────────────────────────────────────────────────────────────┘
                            │
                            │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│  Entra ID    │  │ Fabric API   │  │  Local File  │
│   (OAuth)    │  │  (REST)      │  │   System     │
└──────────────┘  └──────────────┘  └──────────────┘
```

### Data Flow

```
1. User Authentication:
   User → Wails App → Opens Browser → Entra ID → OAuth Token → Store in Keychain

2. Data Sync:
   Background Poller → Fabric API → Parse Response → DuckDB → UI Update

3. User Queries:
   UI → Go Backend → DuckDB Query → Transform Data → Return to UI

4. Notifications:
   Poller Detects Failed Job → Notification Service → OS Native Notification
```

---

## Technology Stack

### Frontend
- **Framework**: Svelte 4.x
- **Build Tool**: Vite 5.x
- **Styling**: Tailwind CSS 3.x
- **Charting**: vis-timeline-graph2d (Gantt charts)
- **Date Handling**: date-fns
- **State Management**: Svelte stores (built-in)
- **HTTP Client**: Native fetch (via Wails bridge)

### Backend
- **Language**: Go 1.21+
- **Desktop Framework**: Wails v2
- **Database**: DuckDB (via go-duckdb)
- **HTTP Client**: net/http (standard library)
- **Authentication**: msal-go (Microsoft Authentication Library)
- **Configuration**: viper
- **Logging**: zap

### Development Tools
- **Package Manager**: npm (frontend), go modules (backend)
- **Linter**: eslint (frontend), golangci-lint (backend)
- **Formatter**: prettier (frontend), gofmt (backend)
- **Testing**: vitest (frontend), testing package (backend)

### Why This Stack?

| Requirement | Technology | Rationale |
|-------------|-----------|-----------|
| Small Binary | Wails | 15MB vs 100MB+ Electron |
| Native Feel | Wails | Real native window, not Chromium |
| Fast UI | Svelte | Compiles to vanilla JS, no virtual DOM |
| Local Database | DuckDB | Embedded, no server, SQL support |
| Auth | MSAL | Official Microsoft library, token caching |
| Cross-Platform | Go + Wails | Single codebase for Win/Mac/Linux |

---

## System Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                         Frontend Layer                       │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  Components/                                                  │
│  ├── Dashboard.svelte          (Main view, job overview)     │
│  ├── GanttChart.svelte         (Timeline visualization)      │
│  ├── JobDetails.svelte         (Drill-down into job)         │
│  ├── Settings.svelte           (App configuration)           │
│  ├── WorkspaceSelector.svelte  (Switch workspaces)           │
│  └── LoginView.svelte          (Authentication UI)           │
│                                                               │
│  Stores/                                                      │
│  ├── auth.js                   (Auth state)                  │
│  ├── jobs.js                   (Job data)                    │
│  ├── workspaces.js             (Workspace data)              │
│  └── settings.js               (App settings)                │
│                                                               │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ Wails Runtime Bridge
                            │
┌─────────────────────────────────────────────────────────────┐
│                         Backend Layer                        │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  cmd/                                                         │
│  └── main.go                   (Application entry point)     │
│                                                               │
│  internal/                                                    │
│  ├── app/                                                     │
│  │   └── app.go               (Main app struct, Wails bind) │
│  │                                                            │
│  ├── auth/                                                    │
│  │   ├── auth.go              (MSAL client wrapper)          │
│  │   ├── token.go             (Token management)             │
│  │   └── cache.go             (Token cache/keychain)         │
│  │                                                            │
│  ├── fabric/                                                  │
│  │   ├── client.go            (Fabric API client)            │
│  │   ├── workspaces.go        (Workspace operations)         │
│  │   ├── pipelines.go         (Pipeline operations)          │
│  │   ├── notebooks.go         (Notebook operations)          │
│  │   └── models.go            (API response models)          │
│  │                                                            │
│  ├── db/                                                      │
│  │   ├── db.go                (DuckDB connection manager)    │
│  │   ├── schema.go            (Table schemas)                │
│  │   ├── queries.go           (SQL queries)                  │
│  │   └── migrations.go        (Schema migrations)            │
│  │                                                            │
│  ├── poller/                                                  │
│  │   ├── poller.go            (Background polling service)   │
│  │   ├── scheduler.go         (Polling intervals)            │
│  │   └── sync.go              (Data synchronization)         │
│  │                                                            │
│  ├── notification/                                            │
│  │   └── notification.go      (OS notifications)             │
│  │                                                            │
│  └── config/                                                  │
│      └── config.go            (App configuration)            │
│                                                               │
└─────────────────────────────────────────────────────────────┘
                            │
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                      External Services                       │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  Microsoft Entra ID          Fabric REST API                 │
│  - OAuth 2.0                 - Workspaces                    │
│  - Token endpoint            - Pipelines                     │
│  - Authorization             - Notebooks                     │
│                              - Job Instances                 │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

---

## Component Specifications

### 1. Authentication Manager (`internal/auth/`)

**Purpose**: Handle Entra ID authentication using MSAL

**Key Responsibilities:**
- Initiate OAuth flow (opens default browser)
- Handle redirect/callback
- Store tokens securely (OS keychain)
- Refresh tokens automatically
- Provide token to API client

**Key Methods:**
```go
type AuthManager struct {
    client      *msal.Client
    tokenCache  *TokenCache
    config      *AuthConfig
}

// Login initiates interactive login flow
func (a *AuthManager) Login() (*Token, error)

// GetToken retrieves cached token or refreshes if expired
func (a *AuthManager) GetToken() (*Token, error)

// Logout clears cached tokens
func (a *AuthManager) Logout() error

// IsAuthenticated checks if valid token exists
func (a *AuthManager) IsAuthenticated() bool
```

**Configuration:**
```go
type AuthConfig struct {
    ClientID     string   // Azure App Registration client ID
    TenantID     string   // Tenant ID or "common"
    RedirectURI  string   // http://localhost:port/callback
    Scopes       []string // ["https://api.fabric.microsoft.com/.default"]
}
```

**Token Storage:**
- **Windows**: Windows Credential Manager
- **macOS**: Keychain
- **Linux**: Secret Service (gnome-keyring/kwallet)

---

### 2. Fabric API Client (`internal/fabric/`)

**Purpose**: Interface with Microsoft Fabric REST APIs

**Key Endpoints:**

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/v1/workspaces` | GET | List all workspaces |
| `/v1/workspaces/{id}/items` | GET | List workspace items |
| `/v1/workspaces/{id}/items/{itemId}/jobs/instances` | GET | Get job runs |
| `/v1/workspaces/{id}/pipelines/{id}/runs` | GET | Pipeline runs |
| `/v1/workspaces/{id}/notebooks/{id}/runs` | GET | Notebook runs |

**Key Types:**
```go
type Client struct {
    httpClient  *http.Client
    authManager *auth.AuthManager
    baseURL     string
}

type Workspace struct {
    ID          string    `json:"id"`
    DisplayName string    `json:"displayName"`
    Type        string    `json:"type"`
    Description string    `json:"description,omitempty"`
}

type JobInstance struct {
    ID              string     `json:"id"`
    JobType         string     `json:"jobType"`
    Status          string     `json:"status"`
    StartTime       time.Time  `json:"startTimeUtc"`
    EndTime         *time.Time `json:"endTimeUtc,omitempty"`
    FailureReason   *string    `json:"failureReason,omitempty"`
    WorkspaceID     string     `json:"workspaceId"`
    ItemID          string     `json:"itemId"`
    ItemDisplayName string     `json:"itemDisplayName"`
}

type PipelineRun struct {
    RunID         string    `json:"runId"`
    PipelineID    string    `json:"pipelineId"`
    PipelineName  string    `json:"pipelineName"`
    Status        string    `json:"status"`
    StartTime     time.Time `json:"startTime"`
    EndTime       *time.Time `json:"endTime,omitempty"`
    DurationMs    *int64    `json:"durationInMs,omitempty"`
}
```

**Key Methods:**
```go
// GetWorkspaces retrieves all accessible workspaces
func (c *Client) GetWorkspaces() ([]Workspace, error)

// GetJobInstances retrieves job runs for a workspace
func (c *Client) GetJobInstances(workspaceID string, startDate, endDate time.Time) ([]JobInstance, error)

// GetPipelineRuns retrieves pipeline runs
func (c *Client) GetPipelineRuns(workspaceID, pipelineID string) ([]PipelineRun, error)

// GetNotebookRuns retrieves notebook runs
func (c *Client) GetNotebookRuns(workspaceID, notebookID string) ([]NotebookRun, error)
```

**Error Handling:**
- Retry logic with exponential backoff
- Rate limit handling (429 responses)
- Token refresh on 401
- Circuit breaker for repeated failures

---

### 3. Database Manager (`internal/db/`)

**Purpose**: Manage DuckDB connection and queries

**Key Responsibilities:**
- Initialize database file
- Run migrations
- Provide query interface
- Handle concurrent access
- Optimize queries

**Key Methods:**
```go
type Database struct {
    conn *sql.DB
    path string
}

// NewDatabase creates or opens database file
func NewDatabase(path string) (*Database, error)

// SaveJobInstances bulk inserts job instances
func (db *Database) SaveJobInstances(jobs []JobInstance) error

// GetJobInstances queries jobs with filters
func (db *Database) GetJobInstances(filter JobFilter) ([]JobInstance, error)

// GetJobStats returns aggregated statistics
func (db *Database) GetJobStats(workspaceID string, from, to time.Time) (*JobStats, error)

// Close closes database connection
func (db *Database) Close() error
```

**Query Optimizations:**
- Indexes on workspace_id, start_time, status
- Materialized views for common aggregations
- Prepared statements for frequent queries
- Batch inserts for sync operations

---

### 4. Background Poller (`internal/poller/`)

**Purpose**: Periodically sync data from Fabric APIs

**Key Responsibilities:**
- Schedule polling intervals
- Fetch new job runs
- Detect changes (new jobs, status updates)
- Trigger notifications on failures
- Handle errors gracefully

**Configuration:**
```go
type PollerConfig struct {
    Interval        time.Duration // Default: 2 minutes
    MaxRetries      int          // Default: 3
    WorkspaceIDs    []string     // Workspaces to monitor
    EnabledTypes    []string     // Job types to track
    NotifyOnFailure bool         // Send notifications
}
```

**Key Methods:**
```go
type Poller struct {
    config     *PollerConfig
    client     *fabric.Client
    db         *db.Database
    notifier   *notification.Service
    ticker     *time.Ticker
    stopChan   chan struct{}
}

// Start begins polling loop
func (p *Poller) Start() error

// Stop gracefully stops polling
func (p *Poller) Stop()

// Poll performs single poll cycle
func (p *Poller) Poll() error

// OnNewData callback when new data is synced
func (p *Poller) OnNewData(callback func([]JobInstance))
```

**Polling Strategy:**
- Initial poll: Last 7 days
- Subsequent polls: Since last poll timestamp
- Incremental sync (only fetch new/updated)
- Exponential backoff on errors

---

### 5. Notification Service (`internal/notification/`)

**Purpose**: Send OS-native desktop notifications

**Key Responsibilities:**
- Show notifications for failed jobs
- Show notifications for long-running jobs (optional)
- Handle user interaction (click to open job details)

**Key Methods:**
```go
type Service struct {
    enabled bool
}

// Notify sends desktop notification
func (s *Service) Notify(title, message string, severity NotificationLevel) error

// NotifyJobFailed sends failure notification
func (s *Service) NotifyJobFailed(job *JobInstance) error

// NotifyLongRunning sends long-running job notification
func (s *Service) NotifyLongRunning(job *JobInstance, threshold time.Duration) error
```

**Notification Levels:**
- `Info`: General updates
- `Warning`: Long-running jobs
- `Error`: Failed jobs, sync errors

---

### 6. Configuration Manager (`internal/config/`)

**Purpose**: Manage application settings

**Configuration File** (`~/.fabric-monitor/config.yaml`):
```yaml
auth:
  client_id: "your-app-registration-id"
  tenant_id: "common"
  redirect_uri: "http://localhost:8080/callback"

polling:
  interval: "2m"
  enabled_workspaces:
    - "workspace-id-1"
    - "workspace-id-2"
  enabled_job_types:
    - "Pipeline"
    - "Notebook"
    - "Dataflow"

notifications:
  enabled: true
  on_failure: true
  on_long_running: false
  long_running_threshold: "30m"

database:
  path: "~/.fabric-monitor/data.duckdb"
  retention_days: 90

ui:
  theme: "dark"
  default_view: "dashboard"
  refresh_interval: "30s"
```

**Key Methods:**
```go
type Config struct {
    Auth          AuthConfig
    Polling       PollingConfig
    Notifications NotificationConfig
    Database      DatabaseConfig
    UI            UIConfig
}

// Load reads configuration from file
func Load(path string) (*Config, error)

// Save writes configuration to file
func (c *Config) Save(path string) error

// Validate checks configuration validity
func (c *Config) Validate() error
```

---

## Database Schema

### Tables

#### 1. `workspaces`
```sql
CREATE TABLE workspaces (
    id VARCHAR PRIMARY KEY,
    display_name VARCHAR NOT NULL,
    type VARCHAR NOT NULL,
    description VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_workspaces_type ON workspaces(type);
```

#### 2. `items`
```sql
CREATE TABLE items (
    id VARCHAR PRIMARY KEY,
    workspace_id VARCHAR NOT NULL REFERENCES workspaces(id),
    display_name VARCHAR NOT NULL,
    type VARCHAR NOT NULL,  -- Pipeline, Notebook, Dataflow, etc.
    description VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_items_workspace ON items(workspace_id);
CREATE INDEX idx_items_type ON items(type);
```

#### 3. `job_instances`
```sql
CREATE TABLE job_instances (
    id VARCHAR PRIMARY KEY,
    workspace_id VARCHAR NOT NULL REFERENCES workspaces(id),
    item_id VARCHAR NOT NULL REFERENCES items(id),
    job_type VARCHAR NOT NULL,
    status VARCHAR NOT NULL,  -- Running, Completed, Failed, Cancelled
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    duration_ms BIGINT,
    failure_reason VARCHAR,
    invoker_type VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_job_instances_workspace ON job_instances(workspace_id);
CREATE INDEX idx_job_instances_item ON job_instances(item_id);
CREATE INDEX idx_job_instances_status ON job_instances(status);
CREATE INDEX idx_job_instances_start_time ON job_instances(start_time);
CREATE INDEX idx_job_instances_composite ON job_instances(workspace_id, start_time, status);
```

#### 4. `pipeline_runs`
```sql
CREATE TABLE pipeline_runs (
    run_id VARCHAR PRIMARY KEY,
    workspace_id VARCHAR NOT NULL REFERENCES workspaces(id),
    pipeline_id VARCHAR NOT NULL REFERENCES items(id),
    pipeline_name VARCHAR NOT NULL,
    status VARCHAR NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    duration_ms BIGINT,
    error_message VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_pipeline_runs_workspace ON pipeline_runs(workspace_id);
CREATE INDEX idx_pipeline_runs_pipeline ON pipeline_runs(pipeline_id);
CREATE INDEX idx_pipeline_runs_start_time ON pipeline_runs(start_time);
```

#### 5. `sync_metadata`
```sql
CREATE TABLE sync_metadata (
    id INTEGER PRIMARY KEY,
    last_sync_time TIMESTAMP NOT NULL,
    sync_type VARCHAR NOT NULL,  -- full, incremental
    records_synced INTEGER NOT NULL,
    errors INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Views

#### `vw_latest_jobs`
```sql
CREATE VIEW vw_latest_jobs AS
SELECT 
    j.id,
    j.workspace_id,
    w.display_name as workspace_name,
    j.item_id,
    i.display_name as item_name,
    i.type as item_type,
    j.status,
    j.start_time,
    j.end_time,
    j.duration_ms,
    j.failure_reason
FROM job_instances j
JOIN workspaces w ON j.workspace_id = w.id
JOIN items i ON j.item_id = i.id
ORDER BY j.start_time DESC;
```

#### `vw_job_stats_daily`
```sql
CREATE VIEW vw_job_stats_daily AS
SELECT 
    DATE_TRUNC('day', start_time) as date,
    workspace_id,
    job_type,
    status,
    COUNT(*) as count,
    AVG(duration_ms) as avg_duration_ms,
    MAX(duration_ms) as max_duration_ms,
    MIN(duration_ms) as min_duration_ms
FROM job_instances
WHERE end_time IS NOT NULL
GROUP BY 1, 2, 3, 4;
```

### Queries

#### Get Jobs for Dashboard
```sql
SELECT 
    id,
    workspace_id,
    item_id,
    job_type,
    status,
    start_time,
    end_time,
    duration_ms,
    failure_reason
FROM job_instances
WHERE workspace_id = ?
    AND start_time >= ?
    AND start_time <= ?
ORDER BY start_time DESC
LIMIT 1000;
```

#### Get Failed Jobs (Last 7 Days)
```sql
SELECT 
    j.id,
    w.display_name as workspace_name,
    i.display_name as item_name,
    j.job_type,
    j.start_time,
    j.failure_reason
FROM job_instances j
JOIN workspaces w ON j.workspace_id = w.id
JOIN items i ON j.item_id = i.id
WHERE j.status = 'Failed'
    AND j.start_time >= NOW() - INTERVAL '7 days'
ORDER BY j.start_time DESC;
```

#### Get Job Success Rate
```sql
SELECT 
    workspace_id,
    job_type,
    COUNT(*) as total_jobs,
    SUM(CASE WHEN status = 'Completed' THEN 1 ELSE 0 END) as successful,
    SUM(CASE WHEN status = 'Failed' THEN 1 ELSE 0 END) as failed,
    ROUND(100.0 * SUM(CASE WHEN status = 'Completed' THEN 1 ELSE 0 END) / COUNT(*), 2) as success_rate
FROM job_instances
WHERE start_time >= ?
GROUP BY workspace_id, job_type;
```

---

## API Integration

### Microsoft Fabric REST API

**Base URL**: `https://api.fabric.microsoft.com/v1`

**Authentication**: Bearer token (from Entra ID)

**Headers**:
```
Authorization: Bearer {token}
Content-Type: application/json
```

### API Endpoints to Implement

#### 1. List Workspaces
```
GET /v1/workspaces
Response: {
  "value": [
    {
      "id": "workspace-guid",
      "displayName": "My Workspace",
      "type": "Workspace",
      "capacityId": "capacity-guid"
    }
  ]
}
```

#### 2. List Workspace Items
```
GET /v1/workspaces/{workspaceId}/items
Response: {
  "value": [
    {
      "id": "item-guid",
      "displayName": "My Pipeline",
      "type": "DataPipeline",
      "workspaceId": "workspace-guid"
    }
  ]
}
```

#### 3. Get Job Instances
```
GET /v1/workspaces/{workspaceId}/items/{itemId}/jobs/instances
Query Parameters:
  - startDateTime: ISO 8601 (optional)
  - endDateTime: ISO 8601 (optional)
  - status: Running|Completed|Failed|Cancelled (optional)

Response: {
  "value": [
    {
      "id": "job-guid",
      "jobType": "Pipeline",
      "status": "Completed",
      "startTimeUtc": "2024-10-15T10:00:00Z",
      "endTimeUtc": "2024-10-15T10:05:00Z",
      "failureReason": null,
      "itemId": "item-guid",
      "itemDisplayName": "My Pipeline"
    }
  ]
}
```

#### 4. Get Pipeline Runs (if available)
```
GET /v1/workspaces/{workspaceId}/pipelines/{pipelineId}/runs
```

### Error Handling

| Status Code | Error | Action |
|-------------|-------|--------|
| 401 | Unauthorized | Refresh token, re-authenticate |
| 403 | Forbidden | Check workspace access |
| 429 | Rate Limited | Exponential backoff |
| 500 | Server Error | Retry with backoff |
| 503 | Service Unavailable | Retry with backoff |

### Rate Limiting Strategy

```go
type RateLimiter struct {
    requests int
    window   time.Duration
    tokens   chan struct{}
}

// Wait blocks until token available
func (rl *RateLimiter) Wait() {
    <-rl.tokens
    go func() {
        time.Sleep(rl.window / time.Duration(rl.requests))
        rl.tokens <- struct{}{}
    }()
}
```

**Limits**:
- 60 requests per minute per workspace
- 1000 requests per hour globally

---

## Authentication Flow

### Initial Login

```
1. User clicks "Login" button
   ↓
2. App opens default browser to Entra ID login page
   - URL: https://login.microsoftonline.com/{tenant}/oauth2/v2.0/authorize
   - Parameters: client_id, redirect_uri, scope, response_type=code
   ↓
3. User authenticates in browser
   ↓
4. Browser redirects to: http://localhost:8080/callback?code=AUTH_CODE
   ↓
5. App's local HTTP server captures code
   ↓
6. App exchanges code for token
   - POST https://login.microsoftonline.com/{tenant}/oauth2/v2.0/token
   - Parameters: client_id, code, redirect_uri, grant_type=authorization_code
   ↓
7. Store access_token and refresh_token in OS keychain
   ↓
8. Close local HTTP server
   ↓
9. Update UI to authenticated state
```

### Token Refresh

```
1. API call returns 401 Unauthorized
   ↓
2. Check if refresh_token exists
   ↓
3. POST https://login.microsoftonline.com/{tenant}/oauth2/v2.0/token
   - Parameters: client_id, refresh_token, grant_type=refresh_token
   ↓
4. Store new access_token (and refresh_token if provided)
   ↓
5. Retry original API call with new token
```

### Token Storage Security

**Windows**:
```go
// Store in Windows Credential Manager
cred := &wincred.Credential{
    TargetName: "FabricMonitor_AccessToken",
    UserName:   userID,
    CredentialBlob: []byte(accessToken),
    Persist:    wincred.PersistLocalMachine,
}
wincred.Write(cred)
```

**macOS**:
```go
// Store in Keychain
keychain.AddItem(
    "FabricMonitor",
    "AccessToken",
    accessToken,
)
```

**Linux**:
```go
// Store via Secret Service (dbus)
service.Store(
    "FabricMonitor",
    "AccessToken",
    accessToken,
)
```

---

## File Structure

```
fabric-monitor/
├── cmd/
│   └── main.go                      # Application entry point
│
├── internal/
│   ├── app/
│   │   └── app.go                   # Main app struct, Wails bindings
│   │
│   ├── auth/
│   │   ├── auth.go                  # MSAL authentication
│   │   ├── token.go                 # Token management
│   │   └── cache.go                 # Secure token storage
│   │
│   ├── fabric/
│   │   ├── client.go                # HTTP client for Fabric API
│   │   ├── workspaces.go            # Workspace API calls
│   │   ├── items.go                 # Items API calls
│   │   ├── jobs.go                  # Job instances API calls
│   │   └── models.go                # API response structs
│   │
│   ├── db/
│   │   ├── db.go                    # DuckDB connection
│   │   ├── schema.go                # Table definitions
│   │   ├── queries.go               # SQL queries
│   │   └── migrations.go            # Schema migrations
│   │
│   ├── poller/
│   │   ├── poller.go                # Background polling service
│   │   ├── scheduler.go             # Polling scheduler
│   │   └── sync.go                  # Data sync logic
│   │
│   ├── notification/
│   │   └── notification.go          # Desktop notifications
│   │
│   └── config/
│       └── config.go                # Configuration management
│
├── frontend/
│   ├── src/
│   │   ├── App.svelte               # Root component
│   │   ├── main.js                  # Entry point
│   │   │
│   │   ├── components/
│   │   │   ├── Dashboard.svelte     # Main dashboard view
│   │   │   ├── GanttChart.svelte    # Timeline visualization
│   │   │   ├── JobList.svelte       # Job list table
│   │   │   ├── JobDetails.svelte    # Job detail modal
│   │   │   ├── FilterBar.svelte     # Filtering controls
│   │   │   ├── WorkspaceSelector.svelte  # Workspace dropdown
│   │   │   ├── Settings.svelte      # Settings panel
│   │   │   └── LoginView.svelte     # Login screen
│   │   │
│   │   ├── stores/
│   │   │   ├── auth.js              # Auth state store
│   │   │   ├── jobs.js              # Jobs data store
│   │   │   ├── workspaces.js        # Workspaces store
│   │   │   └── settings.js          # Settings store
│   │   │
│   │   ├── services/
│   │   │   └── api.js               # Wails API wrapper
│   │   │
│   │   └── utils/
│   │       ├── formatting.js        # Date/time formatting
│   │       └── constants.js         # App constants
│   │
│   ├── public/
│   │   └── favicon.ico
│   │
│   ├── index.html
│   ├── vite.config.js
│   ├── tailwind.config.js
│   └── package.json
│
├── build/
│   ├── appicon.png                  # App icon (1024x1024)
│   ├── windows/
│   │   └── icon.ico
│   ├── darwin/
│   │   └── icon.icns
│   └── linux/
│       └── icon.png
│
├── wails.json                       # Wails configuration
├── go.mod                           # Go dependencies
├── go.sum
├── README.md
└── LICENSE
```

---

## Implementation Phases

### Phase 1: Foundation (Week 1)
**Goal**: Basic app structure, auth, and database

**Tasks**:
1. **Project Setup**
   - [ ] Initialize Wails project
   - [ ] Set up Go modules
   - [ ] Configure Svelte + Tailwind
   - [ ] Set up project structure

2. **Authentication**
   - [ ] Implement MSAL client
   - [ ] Create OAuth flow (browser-based)
   - [ ] Implement token storage (keychain)
   - [ ] Create login UI

3. **Database**
   - [ ] Set up DuckDB connection
   - [ ] Create schema and migrations
   - [ ] Implement basic queries
   - [ ] Test data persistence

**Deliverable**: App that can authenticate and store data locally

---

### Phase 2: API Integration (Week 2)
**Goal**: Fetch and store Fabric data

**Tasks**:
1. **Fabric API Client**
   - [ ] Implement workspace API calls
   - [ ] Implement items API calls
   - [ ] Implement job instances API calls
   - [ ] Add error handling and retries

2. **Data Sync**
   - [ ] Create background poller
   - [ ] Implement incremental sync
   - [ ] Store fetched data in DuckDB
   - [ ] Handle API rate limits

3. **Basic UI**
   - [ ] Create workspace selector
   - [ ] Display job list in table
   - [ ] Show job details on click
   - [ ] Add loading states

**Deliverable**: App that syncs and displays Fabric job data

---

### Phase 3: Visualization (Week 3)
**Goal**: Rich data visualization and filtering

**Tasks**:
1. **Gantt Chart**
   - [ ] Integrate vis-timeline
   - [ ] Map job data to timeline format
   - [ ] Add zoom/pan controls
   - [ ] Color-code by status

2. **Dashboard**
   - [ ] Summary statistics cards
   - [ ] Success rate charts
   - [ ] Failed jobs list
   - [ ] Long-running jobs alerts

3. **Filtering**
   - [ ] Date range picker
   - [ ] Status filter
   - [ ] Job type filter
   - [ ] Workspace filter
   - [ ] Search by job name

**Deliverable**: Full-featured dashboard with visualizations

---

### Phase 4: Polish & Features (Week 4)
**Goal**: Notifications, settings, and final touches

**Tasks**:
1. **Notifications**
   - [ ] Desktop notifications for failures
   - [ ] Long-running job alerts
   - [ ] Notification settings

2. **Settings Panel**
   - [ ] Polling interval configuration
   - [ ] Workspace selection
   - [ ] Theme toggle (light/dark)
   - [ ] Notification preferences

3. **Polish**
   - [ ] Error handling UI
   - [ ] Empty states
   - [ ] Loading animations
   - [ ] Keyboard shortcuts
   - [ ] App icon and branding

4. **Build & Distribution**
   - [ ] Build for Windows
   - [ ] Build for macOS
   - [ ] Build for Linux
   - [ ] Create installers
   - [ ] Write documentation

**Deliverable**: Production-ready application

---

## Development Setup

### Prerequisites

1. **Go 1.21+**
   ```bash
   # macOS
   brew install go
   
   # Windows
   winget install GoLang.Go
   
   # Linux
   sudo snap install go --classic
   ```

2. **Node.js 18+**
   ```bash
   # macOS
   brew install node
   
   # Windows
   winget install OpenJS.NodeJS
   
   # Linux
   curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
   sudo apt-get install -y nodejs
   ```

3. **Wails CLI**
   ```bash
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   ```

4. **Platform-Specific Tools**

   **Windows**:
   - Install Visual Studio Build Tools
   - WebView2 Runtime (usually pre-installed on Windows 10/11)

   **macOS**:
   - Xcode Command Line Tools
   ```bash
   xcode-select --install
   ```

   **Linux**:
   - WebKit2GTK development libraries
   ```bash
   # Debian/Ubuntu
   sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev
   
   # Fedora
   sudo dnf install gtk3-devel webkit2gtk3-devel
   ```

### Initial Setup

```bash
# Create project
wails init -n fabric-monitor -t svelte

cd fabric-monitor

# Install frontend dependencies
cd frontend
npm install

# Install additional dependencies
npm install -D tailwindcss postcss autoprefixer
npm install vis-timeline date-fns

# Initialize Tailwind
npx tailwindcss init -p

cd ..

# Install Go dependencies
go get github.com/marcboeker/go-duckdb
go get github.com/AzureAD/microsoft-authentication-library-for-go
go get github.com/spf13/viper
go get go.uber.org/zap
```

### Running in Development

```bash
# Development mode (hot reload)
wails dev

# Build for production
wails build

# Build for specific platform
wails build -platform windows/amd64
wails build -platform darwin/arm64
wails build -platform linux/amd64
```

### Environment Variables

Create `.env` file:
```bash
# Azure App Registration
AZURE_CLIENT_ID=your-client-id-here
AZURE_TENANT_ID=common
AZURE_REDIRECT_URI=http://localhost:8080/callback

# Development
DEBUG=true
LOG_LEVEL=debug
```

---

## UI/UX Specifications

### Design System

**Colors** (Tailwind):
```javascript
// tailwind.config.js
module.exports = {
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#eff6ff',
          500: '#3b82f6',
          900: '#1e3a8a',
        },
        success: '#10b981',
        warning: '#f59e0b',
        error: '#ef4444',
        running: '#3b82f6',
        completed: '#10b981',
        failed: '#ef4444',
      },
    },
  },
}
```

**Typography**:
- Font: Inter (system font fallback)
- Headers: font-semibold
- Body: font-normal
- Code: font-mono

### Component Specifications

#### 1. Dashboard View

**Layout**:
```
┌─────────────────────────────────────────────────────────┐
│  Fabric Monitor                    [Workspace ▼] [⚙️]   │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐        │
│  │ Total Jobs │  │  Success   │  │  Failed    │        │
│  │    1,234   │  │   98.5%    │  │     18     │        │
│  └────────────┘  └────────────┘  └────────────┘        │
│                                                          │
│  [Date Range Picker] [Status: All ▼] [Type: All ▼]     │
│                                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │          Gantt Chart (Timeline View)               │ │
│  │  ════════════════════════════════════════════════  │ │
│  │  Pipeline A  ▓▓▓▓▓░░░░                            │ │
│  │  Pipeline B      ▓▓▓▓▓▓▓▓                         │ │
│  │  Notebook X  ▓▓░░░░                               │ │
│  │  ════════════════════════════════════════════════  │ │
│  └────────────────────────────────────────────────────┘ │
│                                                          │
│  Recent Failed Jobs                                     │
│  ┌────────────────────────────────────────────────────┐ │
│  │ ❌ Pipeline A | 2:34 PM | Connection timeout       │ │
│  │ ❌ Notebook B | 1:15 PM | Out of memory            │ │
│  └────────────────────────────────────────────────────┘ │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

**Features**:
- Real-time updates (30s refresh)
- Click job to view details
- Filter by date, status, type
- Export to CSV

#### 2. Job Details Modal

```
┌─────────────────────────────────────────────┐
│  Job Details                           [✕]  │
├─────────────────────────────────────────────┤
│                                              │
│  Pipeline: Daily Sales ETL                  │
│  Status: ❌ Failed                          │
│  Duration: 5m 23s                           │
│                                              │
│  Started:  Oct 15, 2024 10:00:00 AM        │
│  Ended:    Oct 15, 2024 10:05:23 AM        │
│                                              │
│  Error:                                     │
│  ┌──────────────────────────────────────┐  │
│  │ Connection timeout after 30s         │  │
│  │ Failed to connect to SQL Server      │  │
│  └──────────────────────────────────────┘  │
│                                              │
│  [View in Fabric]  [Copy Error]            │
│                                              │
└─────────────────────────────────────────────┘
```

#### 3. Settings Panel

```
┌─────────────────────────────────────────────┐
│  Settings                              [✕]  │
├─────────────────────────────────────────────┤
│                                              │
│  Workspaces                                 │
│  ☑ Workspace 1                              │
│  ☑ Workspace 2                              │
│  ☐ Workspace 3 (disabled)                   │
│                                              │
│  Polling                                    │
│  Interval: [2 minutes ▼]                    │
│                                              │
│  Notifications                               │
│  ☑ Enable desktop notifications             │
│  ☑ Notify on job failure                    │
│  ☐ Notify on long-running (>30min)          │
│                                              │
│  Data                                        │
│  Retention: [90 days ▼]                     │
│  [Clear Local Data]                         │
│                                              │
│  Account                                     │
│  Logged in as: user@company.com             │
│  [Logout]                                    │
│                                              │
│         [Cancel]        [Save]              │
│                                              │
└─────────────────────────────────────────────┘
```

### Responsive Behavior

- Minimum window size: 1200x800
- Gantt chart scrolls horizontally
- Job list scrolls vertically
- Sidebar collapsible on smaller screens

---

## Testing Strategy

### Unit Tests

**Go Backend**:
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/db
```

**Test Coverage Targets**:
- Auth: 85%
- Fabric client: 90%
- Database: 95%
- Poller: 80%

**Example Test**:
```go
func TestAuthManager_GetToken(t *testing.T) {
    // Setup
    mockCache := &MockTokenCache{}
    am := &AuthManager{tokenCache: mockCache}
    
    // Test cached token
    mockCache.token = &Token{AccessToken: "cached"}
    token, err := am.GetToken()
    assert.NoError(t, err)
    assert.Equal(t, "cached", token.AccessToken)
    
    // Test expired token
    mockCache.token = &Token{ExpiresAt: time.Now().Add(-1 * time.Hour)}
    token, err = am.GetToken()
    assert.Error(t, err)
}
```

**Frontend**:
```bash
# Run Vitest
npm test

# Run with coverage
npm test -- --coverage
```

### Integration Tests

**Database Integration**:
```go
func TestDatabase_SaveAndRetrieveJobs(t *testing.T) {
    // Create temp database
    db, err := NewDatabase(":memory:")
    require.NoError(t, err)
    defer db.Close()
    
    // Insert jobs
    jobs := []JobInstance{
        {ID: "1", Status: "Completed"},
        {ID: "2", Status: "Failed"},
    }
    err = db.SaveJobInstances(jobs)
    require.NoError(t, err)
    
    // Retrieve
    retrieved, err := db.GetJobInstances(JobFilter{})
    require.NoError(t, err)
    assert.Len(t, retrieved, 2)
}
```

**API Integration** (requires test credentials):
```go
func TestFabricClient_GetWorkspaces(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    client := NewClient(testToken)
    workspaces, err := client.GetWorkspaces()
    require.NoError(t, err)
    assert.NotEmpty(t, workspaces)
}
```

### End-to-End Tests

Use Wails' built-in testing or Playwright for E2E:

```javascript
// e2e/login.spec.js
test('user can login', async ({ page }) => {
    await page.goto('http://localhost:34115');
    await page.click('button:has-text("Login")');
    // Browser window opens
    // After auth, check for dashboard
    await expect(page.locator('h1')).toContainText('Dashboard');
});
```

### Manual Testing Checklist

**Before Each Release**:
- [ ] Login flow works
- [ ] Workspace selector populates
- [ ] Jobs load in dashboard
- [ ] Gantt chart renders correctly
- [ ] Filtering works
- [ ] Job details modal displays
- [ ] Settings save/load
- [ ] Notifications appear
- [ ] Background polling works
- [ ] Token refresh works
- [ ] App updates work
- [ ] Test on Windows
- [ ] Test on macOS
- [ ] Test on Linux

---

## Deployment Strategy

### Build Configuration

**wails.json**:
```json
{
  "name": "fabric-monitor",
  "outputfilename": "FabricMonitor",
  "frontend:install": "npm install",
  "frontend:build": "npm run build",
  "author": {
    "name": "Your Name",
    "email": "you@example.com"
  },
  "info": {
    "companyName": "Your Company",
    "productName": "Fabric Monitor",
    "productVersion": "1.0.0",
    "copyright": "Copyright © 2024",
    "comments": "Monitor Microsoft Fabric workspaces"
  }
}
```

### Build Commands

```bash
# Windows
wails build -platform windows/amd64 -webview2 embed

# macOS (Universal binary)
wails build -platform darwin/universal

# Linux
wails build -platform linux/amd64
```

### Distribution

#### Windows
- **Output**: `FabricMonitor.exe`
- **Size**: ~15-20MB
- **Installer**: Use Inno Setup or NSIS
- **Auto-update**: GitHub Releases

**Inno Setup Script**:
```iss
[Setup]
AppName=Fabric Monitor
AppVersion=1.0.0
DefaultDirName={pf}\FabricMonitor
DefaultGroupName=Fabric Monitor
OutputDir=dist
OutputBaseFilename=FabricMonitor-Setup

[Files]
Source: "build\bin\FabricMonitor.exe"; DestDir: "{app}"

[Icons]
Name: "{group}\Fabric Monitor"; Filename: "{app}\FabricMonitor.exe"
Name: "{commondesktop}\Fabric Monitor"; Filename: "{app}\FabricMonitor.exe"
```

#### macOS
- **Output**: `FabricMonitor.app`
- **Size**: ~18-22MB
- **Distribution**: DMG file
- **Code Signing**: Required for Gatekeeper

```bash
# Create DMG
create-dmg \
  --volname "Fabric Monitor" \
  --window-pos 200 120 \
  --window-size 800 400 \
  --icon-size 100 \
  --icon "FabricMonitor.app" 200 190 \
  --hide-extension "FabricMonitor.app" \
  --app-drop-link 600 185 \
  "FabricMonitor-1.0.0.dmg" \
  "build/bin/"
```

#### Linux
- **Output**: `FabricMonitor`
- **Size**: ~16-20MB
- **Distribution**: AppImage, .deb, .rpm
- **Desktop Integration**: .desktop file

**.desktop file**:
```ini
[Desktop Entry]
Name=Fabric Monitor
Exec=/usr/bin/fabric-monitor
Icon=fabric-monitor
Type=Application
Categories=Utility;Development;
```

### Auto-Update

Use **Wails auto-update** or **Tauri updater**:

```go
// Check for updates on startup
func (a *App) CheckForUpdates() (*UpdateInfo, error) {
    resp, err := http.Get("https://api.github.com/repos/user/fabric-monitor/releases/latest")
    if err != nil {
        return nil, err
    }
    
    var release GitHubRelease
    json.NewDecoder(resp.Body).Decode(&release)
    
    if release.TagName != currentVersion {
        return &UpdateInfo{
            Version:     release.TagName,
            DownloadURL: release.Assets[0].BrowserDownloadURL,
        }, nil
    }
    
    return nil, nil
}
```

### Release Checklist

**Pre-Release**:
- [ ] Run full test suite
- [ ] Update version in `wails.json`
- [ ] Update CHANGELOG.md
- [ ] Build for all platforms
- [ ] Test installers on clean VMs
- [ ] Code sign (macOS/Windows)

**Release**:
- [ ] Create Git tag
- [ ] Push to GitHub
- [ ] Upload binaries to GitHub Releases
- [ ] Update download links in README
- [ ] Announce on relevant channels

**Post-Release**:
- [ ] Monitor for crash reports
- [ ] Respond to issues
- [ ] Plan next version

---

## Security Considerations

### Data Protection
1. **Tokens**: Stored in OS keychain (encrypted)
2. **Database**: Local file, no network exposure
3. **API Keys**: Never hardcoded, use config file

### Network Security
1. **HTTPS Only**: All API calls use TLS
2. **Token Validation**: Verify token expiry before use
3. **Rate Limiting**: Prevent API abuse

### Code Security
1. **Input Validation**: Sanitize all user inputs
2. **SQL Injection**: Use parameterized queries
3. **XSS Protection**: Svelte auto-escapes by default

### Privacy
1. **No Telemetry**: Optional analytics with opt-in
2. **Local Data**: Everything stored locally
3. **No Cloud Sync**: User controls data

---

## Performance Targets

| Metric | Target | Measurement |
|--------|--------|-------------|
| Initial Load | < 2s | Time to dashboard visible |
| Data Sync | < 5s | 1000 jobs synced |
| Query Response | < 100ms | DuckDB queries |
| Memory Usage | < 150MB | Idle state |
| Binary Size | < 20MB | All platforms |
| CPU Usage | < 5% | Idle with polling |

---

## Monitoring & Observability

### Logging

```go
// Initialize logger
logger, _ := zap.NewProduction()
defer logger.Sync()

// Log levels
logger.Info("App started", zap.String("version", "1.0.0"))
logger.Error("API call failed", zap.Error(err))
logger.Debug("Polling workspace", zap.String("id", workspaceID))
```

**Log Locations**:
- **Windows**: `%APPDATA%\FabricMonitor\logs\`
- **macOS**: `~/Library/Logs/FabricMonitor/`
- **Linux**: `~/.local/share/FabricMonitor/logs/`

### Crash Reporting

Optional integration with Sentry:
```go
sentry.Init(sentry.ClientOptions{
    Dsn: "https://your-dsn@sentry.io/project",
    Environment: "production",
    Release: "fabric-monitor@1.0.0",
})
```

---

## Future Enhancements

### Phase 2 Features (Post-MVP)
1. **Export Reports**: PDF/Excel export of job stats
2. **Custom Dashboards**: User-defined dashboard layouts
3. **Alerting Rules**: Custom rules for notifications
4. **Capacity Metrics**: Track capacity usage
5. **Collaboration**: Share dashboards with team
6. **Mobile App**: iOS/Android companion app
7. **Slack Integration**: Post notifications to Slack
8. **Advanced Analytics**: ML-powered insights

### Technical Improvements
1. **Performance**: Virtualized lists for 10k+ jobs
2. **Offline Mode**: Full offline functionality
3. **Multi-Tenant**: Support multiple Azure tenants
4. **Data Export**: Export to Parquet, JSON
5. **Plugin System**: Extensible architecture

---

## Getting Started (Quick Start)

```bash
# 1. Clone/create project
wails init -n fabric-monitor -t svelte
cd fabric-monitor

# 2. Install dependencies
cd frontend && npm install
cd .. && go mod tidy

# 3. Set up Azure App Registration
# - Go to portal.azure.com
# - Create App Registration
# - Add redirect URI: http://localhost:8080/callback
# - Add API permission: Fabric.ReadWrite.All
# - Copy Client ID

# 4. Configure app
# Create config.yaml with your client ID

# 5. Run in dev mode
wails dev

# 6. Build for production
wails build
```

---

## Support & Documentation

### Resources
- **Wails Docs**: https://wails.io
- **Fabric API Docs**: https://learn.microsoft.com/fabric/
- **DuckDB Docs**: https://duckdb.org/docs/
- **Svelte Docs**: https://svelte.dev/docs

### Community
- GitHub Issues: Bug reports and feature requests
- Discussions: Questions and community support

---

## Conclusion

This implementation plan provides a complete blueprint for building a production-ready Microsoft Fabric monitoring desktop application. The architecture leverages modern technologies (Wails, Svelte, DuckDB) to deliver a fast, lightweight, and native user experience.

**Key Strengths**:
- ✅ Desktop-native (no hosting required)
- ✅ Local-first data storage
- ✅ Native authentication flow
- ✅ Cross-platform compatibility
- ✅ Small binary size (~15MB)
- ✅ Production-ready architecture

**Next Steps**:
1. Review and approve this plan
2. Set up development environment
3. Create Azure App Registration
4. Begin Phase 1 implementation
5. Iterate based on feedback

Ready to build! 🚀
