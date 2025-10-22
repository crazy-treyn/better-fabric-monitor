---
mode: agent
---
# Bump Version Number

## Objective
Update the Better Fabric Monitor application to a new version number and create a git tag that will trigger an automated build and release via GitHub Actions.

## Instructions for LLM Agent

**Before starting, ask the user for the following information:**

1. **New version number** (format: X.Y.Z, e.g., 0.3.0, 1.0.0, etc.)
2. **Release description** (brief summary of what's new in this version for the git tag message)

Once you have this information, proceed with the following steps in order:

---

## Step 1: Update Version in Files

### File 1: `wails.json`
Update the `info.productVersion` field:
```json
"info": {
  "productVersion": "[NEW_VERSION]"
}
```

### File 2: `internal/config/config.go`
Update the default version in the `Load()` function (around line 98):
```go
viper.SetDefault("app.version", "[NEW_VERSION]")
```

---

## Step 2: Git Operations

### 2.1 Stage the changes
```powershell
git add wails.json internal/config/config.go
```

### 2.2 Commit with version message
```powershell
git commit -m "Bump version to [NEW_VERSION]"
```

### 2.3 Create annotated git tag
```powershell
git tag -a v[NEW_VERSION] -m "Release v[NEW_VERSION] - [RELEASE_DESCRIPTION]"
```

---

## Step 3: Push to Repository

Push the version changes and tag to trigger the automated build:

```powershell
git push origin [CURRENT_BRANCH]
git push origin v[NEW_VERSION]
```

**GitHub Actions will automatically:**
1. Build the production executable with the new version embedded
2. Generate the SHA-256 checksum file
3. Create a GitHub Release with both files attached
4. Add installation instructions and verification steps

---

## Step 4: Verification

After pushing, verify the following and **report these results back to the user**:

1. **GitHub Actions workflow:** Check that the workflow started at `https://github.com/crazy-treyn/better-fabric-monitor/actions`
2. **Workflow completion:** Wait for the build to complete (typically 2-3 minutes)
3. **Release created:** Verify release exists at `https://github.com/crazy-treyn/better-fabric-monitor/releases/tag/v[NEW_VERSION]`
4. **Artifacts attached:** Confirm both `better-fabric-monitor.exe` and `better-fabric-monitor.exe.sha256` are attached to the release

**Present the verification results to the user with links to:**
- GitHub Actions workflow run
- GitHub Release page

---

## Summary of Files Modified

- `wails.json` - Primary version metadata (embedded in Windows EXE)
- `internal/config/config.go` - Backend version (displayed in UI)

## Automated Build Process

The `.github/workflows/release.yml` workflow handles:
- ✅ Installing Go 1.24+
- ✅ Installing Node.js 24+
- ✅ Installing Wails CLI
- ✅ Installing frontend dependencies
- ✅ Building production executable
- ✅ Generating SHA-256 checksum
- ✅ Creating GitHub Release with artifacts
- ✅ Adding installation instructions

## Build Artifacts (Attached to GitHub Release)

- `better-fabric-monitor.exe` - Production executable with embedded version
- `better-fabric-monitor.exe.sha256` - SHA-256 checksum for integrity verification

Users can download from: `https://github.com/crazy-treyn/better-fabric-monitor/releases/latest`

---

## Version Flow Diagram

```
wails.json (productVersion)
    ↓
Windows EXE metadata (file properties)

internal/config/config.go (app.version)
    ↓
Go backend API (GetAppVersion)
    ↓
Frontend UI (Logs page footer: "• v[NEW_VERSION]")
```

