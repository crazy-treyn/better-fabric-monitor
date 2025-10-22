# GitHub Actions Workflows

This project uses two separate workflows following industry best practices:

## 🔨 Build Workflow (`build.yml`)

**Purpose:** Continuous Integration - Verify builds work on every commit

**Triggers:**
- Every push to any branch (except tags)
- Pull requests to `main`

**Actions:**
- ✅ Installs dependencies (Go, Node.js, Wails)
- ✅ Builds the application
- ✅ Verifies executable was created
- ✅ Uploads build artifact (available for 7 days)

**Access artifacts:**
1. Go to Actions tab
2. Click on the workflow run
3. Download from "Artifacts" section at bottom

**Status badge:**
```markdown
![Build](https://github.com/crazy-treyn/better-fabric-monitor/actions/workflows/build.yml/badge.svg)
```

---

## 🚀 Release Workflow (`release.yml`)

**Purpose:** Automated releases - Create GitHub Releases with binaries

**Triggers:**
- Only when a version tag is pushed (e.g., `v0.2.0`, `v1.0.0`)

**Actions:**
- ✅ Builds production executable
- ✅ Generates SHA-256 checksum
- ✅ Creates GitHub Release
- ✅ Attaches `.exe` and `.sha256` files
- ✅ Adds installation instructions

**Usage:**
```powershell
# From main branch only
git tag -a v0.2.0 -m "Release v0.2.0 - Description"
git push origin v0.2.0
```

**Result:**
- Release appears at: https://github.com/crazy-treyn/better-fabric-monitor/releases

---

## Workflow Comparison

| Feature | Build | Release |
|---------|-------|---------|
| **When** | Every commit/PR | Version tags only |
| **From** | Any branch | Any branch (recommend main) |
| **Creates Release** | ❌ No | ✅ Yes |
| **Artifacts** | Temporary (7 days) | Permanent (GitHub Releases) |
| **Purpose** | Verify builds work | Official distribution |

---

## Testing the Build Process

### Option 1: Push to Feature Branch
```powershell
git add .
git commit -m "Test build"
git push origin feature/my-feature
```
→ `build.yml` runs automatically

### Option 2: Create Pull Request
Create a PR to `main` → `build.yml` runs on every commit

### Option 3: Local Build (Fastest)
```powershell
wails build
```

---

## Best Practices

✅ **DO:**
- Let `build.yml` catch build issues early on feature branches
- Only create release tags from `main` branch
- Use semantic versioning (v1.2.3)
- Test locally before pushing tags

❌ **DON'T:**
- Push version tags from feature branches
- Skip the build workflow by pushing directly to main
- Reuse or delete tags (creates confusion)

---

## Build Requirements

Both workflows install:
- Go 1.24+
- Node.js 24+
- Wails CLI (latest)
- Frontend dependencies (npm)

Build time: ~2-3 minutes
