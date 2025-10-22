---
mode: agent
---
# Bump Version Number

## Objective
Update the Better Fabric Monitor application to a new version number across all relevant files, create a git tag, sign the executable, and prepare for release.

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

## Step 3: Build Production Executable

Build the production executable with the new version embedded:

```powershell
wails build
```

The executable will be created at: `build/bin/better-fabric-monitor.exe`

---

## Step 4: Generate Signature File

Create a SHA-256 signature file for the executable:

```powershell
Get-FileHash -Algorithm SHA256 build/bin/better-fabric-monitor.exe | Select-Object -ExpandProperty Hash | Out-File -Encoding ASCII build/bin/better-fabric-monitor.exe.sha256
```

This creates a signature file that can be used to verify the integrity of the executable.

---

## Step 5: Push to Repository

Push the version changes and tag to the remote repository:

```powershell
git push origin [CURRENT_BRANCH]
git push origin v[NEW_VERSION]
```

---

## Step 6: Verification

Verify the following and **report these results back to the user**:

1. **Executable metadata:** Right-click `build/bin/better-fabric-monitor.exe` → Properties → Details tab should show version [NEW_VERSION]
2. **Signature file exists:** Check that `build/bin/better-fabric-monitor.exe.sha256` was created
3. **UI displays version:** Run the app and navigate to Logs page - footer should show "• v[NEW_VERSION]"
4. **Git tag exists:** Run `git tag -l` and confirm `v[NEW_VERSION]` is listed

**Present the verification results to the user in a clear summary showing which items passed/failed.**

---

## Summary of Files Modified

- `wails.json` - Primary version metadata (embedded in Windows EXE)
- `internal/config/config.go` - Backend version (displayed in UI)

## Build Artifacts

- `build/bin/better-fabric-monitor.exe` - Signed production executable
- `build/bin/better-fabric-monitor.exe.sha256` - SHA-256 signature for verification

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

