---
mode: agent
---
# Release Workflow (Automated with GitHub MCP)

## Objective
Automated end-to-end release workflow for Better Fabric Monitor using GitHub MCP server integration, with human approval gates at critical steps.

## Prerequisites
- GitHub MCP server connected to VS Code
- Authenticated with GitHub
- Working on the Better Fabric Monitor repository

---

## Instructions for LLM Agent

This workflow automates the entire release process from version bump to GitHub Release creation. **Ask for user approval at each major step before proceeding.**

### **Step 0: Gather Information**

Ask the user for:
1. **New version number** (format: X.Y.Z, e.g., 0.3.0, 1.0.0)
2. **Confirmation that everything is merged into main in GitHub**: This workflow assumes that the code in `main` is going to be released.

**Note:** You'll write the full release notes later in GitHub's draft editor, where you can:
- Add formatted text with GitHub's markdown editor
- Drag & drop screenshots and images
- Preview how it will look
- Take your time to make it perfect

---

## Phase 1: Version Bump (Feature Branch)

### **1.1 Check Current State**

```javascript
// Use GitHub MCP to check current branch and repository state
- Get current branch name
- Check local git status
```

**Tools:**
- `mcp_github_github_get_me` - Verify GitHub authentication
- Terminal commands for git status

### **1.2 Update Main Branch**

**Ensure local main is up to date:**

```powershell
# Save current branch
$currentBranch = git branch --show-current

# Switch to main and pull latest
git checkout main
git pull origin main

# Switch back to original branch if not on main
if ($currentBranch -ne "main") {
    git checkout $currentBranch
}
```

**Human Approval Gate:** 
> "Local `main` branch updated to latest. Current state:
> - Latest commit: [COMMIT_HASH]
> - Ready to create release branch from latest main
> 
> Proceed?"

### **1.3 Create Feature Branch from Main**

```powershell
# Create new branch from updated main
git checkout -b release/v[VERSION] main
```

**Verify:**
- Branch created successfully
- Based on latest main commit

**Human Approval Gate:** 
> "Created feature branch `release/v[VERSION]` from latest main. Continue with version updates?"

### **1.4 Update Version Files**

Update these files:
1. `wails.json` → `info.productVersion`
2. `internal/config/config.go` → `viper.SetDefault("app.version", "[VERSION]")`

**Tools:**
- `replace_string_in_file` for both files

### **1.5 Commit Changes**

```powershell
git add wails.json internal/config/config.go
git commit -m "Bump version to [VERSION]"
git push origin release/v[VERSION]
```

**Human Approval Gate:**
> "Version files updated. Push to branch `release/v[VERSION]`?"

---

## Phase 2: Pull Request Creation

### **2.1 Create Pull Request via GitHub MCP**

**Tools:**
- `mcp_github_github_create_pull_request`

**Parameters:**
```javascript
{
  owner: "crazy-treyn",
  repo: "better-fabric-monitor",
  title: "Release v[VERSION]",
  head: "release/v[VERSION]",
  base: "main",
  body: `## Release v[VERSION]

Version bump for upcoming release.

### Files Modified
- \`wails.json\` - Updated productVersion to [VERSION]
- \`internal/config/config.go\` - Updated app.version to [VERSION]

### Checklist
- [x] Version bumped in all required files
- [ ] PR reviewed and approved
- [ ] Ready to merge and release

---

**Next Steps:**
1. Review and approve this PR
2. Merge to \`main\`
3. Agent will create tag and trigger release workflow
4. You'll write release notes in GitHub's draft editor
`
}
```

**Human Approval Gate:**
> "Pull Request created: [PR_URL]. Review the PR and let me know when you're ready to merge."

---

## Phase 3: Wait for User Merge

**Inform the user:**
```
✅ Version bump PR created successfully!

📝 PR #[NUMBER]: Release v[VERSION]
🔗 Review at: [PR_URL]

⏸️ PAUSED: Please review and merge the PR when ready.

After merging, say "PR merged" or "ready to release" to continue.
```

**Agent should WAIT for user confirmation before proceeding.**

---

## Phase 4: Create Release (After PR Merged)

### **4.1 Verify Merge and Switch to Main**

**Ensure we're on main with latest changes:**

```powershell
# Switch to main (even if we're on release branch)
git checkout main

# Pull latest changes (includes merged PR)
git pull origin main
```

**Tools:**
- Terminal commands

**Human Approval Gate:**
> "Switched to main and pulled latest changes. Ready to verify version [VERSION] is in the code?"

### **4.2 Verify Version in Code**

Read and confirm:
- `wails.json` shows version [VERSION]
- `internal/config/config.go` shows version [VERSION]

**Tools:**
- `read_file` for both files

### **4.3 Check Existing Tags**

```javascript
// Use GitHub MCP to check existing tags
mcp_github_github_list_tags({
  owner: "crazy-treyn",
  repo: "better-fabric-monitor"
})
```

**Verify:** Tag `v[VERSION]` does not already exist

### **4.4 Create Git Tag**

```powershell
git tag -a v[VERSION] -m "Release v[VERSION]"
```

**Note:** The tag message is kept simple - you'll write full release notes in GitHub

**Human Approval Gate:**
> "Tag v[VERSION] created locally. Ready to push and trigger release workflow? This will:
> - Push tag to GitHub
> - Trigger automated build (~2-3 min)
> - Create GitHub Release with binaries
> 
> Proceed?"

### **4.5 Push Tag**

```powershell
git push origin v[VERSION]
```

---

## Phase 5: Monitor Release Workflow

### **5.1 Inform User and Pause**

**Inform user:**
```
🚀 Release workflow triggered!

⏱️ Build in progress (~2-3 minutes)
📊 Monitor at: https://github.com/crazy-treyn/better-fabric-monitor/actions

⏸️ PAUSED: Please check the workflow status and confirm when complete.

When the build finishes (green checkmark), say "build complete" or "workflow done" to continue.
```

**Agent should WAIT for user confirmation before proceeding.**

---

### **5.2 Verify Release Created**

After user confirms workflow completed, use GitHub MCP to verify:

```javascript
// Check if release exists
mcp_github_github_get_release_by_tag({
  owner: "crazy-treyn",
  repo: "better-fabric-monitor",
  tag: "v[VERSION]"
})
```

**Verify (if release exists):**
- ✅ Release exists (as **draft**)
- ✅ Has attached assets (2 files expected)
- ✅ Release notes generated

**Inform user:**
```
✅ Draft release created successfully!

📦 Draft Release Page:
https://github.com/crazy-treyn/better-fabric-monitor/releases/tag/v[VERSION]

📝 Next: Write Your Release Notes

The release is currently a DRAFT with binaries already attached. Now it's time to write your release notes!

**Step-by-Step Guide:**

1. **Open the draft release:**
   Click the link above or go to:
   https://github.com/crazy-treyn/better-fabric-monitor/releases
   
2. **Click "Edit" button** (pencil icon on the right)

3. **Write your release notes** in the editor. Suggested format:
   ```markdown
   ## What's New in v[VERSION]
   
   ### ✨ Features
   - New analytics dashboard with real-time metrics
   - Enhanced filtering capabilities
   
   ### 🐛 Bug Fixes
   - Fixed UTC timestamp display issues
   - Resolved caching problems
   
   ### 📸 Screenshots
   [Drag & drop your images here]
   ```

4. **Add screenshots/images:**
   - Drag images from your computer directly into the editor
   - They'll auto-upload and insert as: `![image](url)`
   - You can add captions below each image

5. **Preview your release:**
   - Click the "Preview" tab to see how it looks
   - Switch back to "Write" to edit

6. **Auto-generated changelog:**
   - GitHub automatically adds a "Full Changelog" section
   - Shows all commits since last release
   - You can keep or remove this

7. **When you're happy with it:**
   - Click "Publish release" at the bottom
   - This makes it publicly visible

📎 Already attached:
- better-fabric-monitor.exe (~59 MB) ✅
- better-fabric-monitor.exe.sha256 (~65 bytes) ✅

💡 Tips:
- Take your time - drafts are saved automatically
- You can come back and edit anytime before publishing
- Use markdown for formatting (bold, lists, headers, etc.)
- Test the download links in the draft if you want

Let me know when you've published the release and we'll clean up!
```

**If release not found:**
- Check if workflow actually succeeded
- Report error details
- Provide Actions URL for troubleshooting
- Skip to cleanup phase

---

## Phase 6: Final Verification & Cleanup

### **6.1 Report Results**

**Generate summary report:**
```
✅ Release v[VERSION] Created as Draft!

📦 Release Page:
https://github.com/crazy-treyn/better-fabric-monitor/releases/tag/v[VERSION]

📎 Artifacts:
- better-fabric-monitor.exe (~59 MB) ✅
- better-fabric-monitor.exe.sha256 (~65 bytes) ✅

� Next Steps:
1. Review the draft release
2. Add screenshots/images if needed (drag & drop)
3. Edit release notes if needed
4. Click "Publish release" to make it public

✅ Verification Checklist:
- [x] Version bumped in code
- [x] PR created and merged
- [x] Tag created and pushed
- [x] Release workflow succeeded
- [x] Binaries attached to release
- [ ] Release published (manual step)

🧹 Cleanup (optional):
- Delete feature branch: git branch -d release/v[VERSION]
- Delete remote branch: git push origin --delete release/v[VERSION]
```

### **6.2 Optional: Delete Feature Branch**

**Human Approval Gate:**
> "Release successful! Delete the feature branch `release/v[VERSION]`?"

If yes:
```powershell
# Ensure we're not on the branch we're deleting
git checkout main

# Delete local branch
git branch -d release/v[VERSION]

# Delete remote branch
git push origin --delete release/v[VERSION]
```

**Note:** `-d` flag ensures branch is fully merged before deleting (safe delete)

---

## Error Handling

### **If Build Fails:**
1. Check Actions logs: https://github.com/crazy-treyn/better-fabric-monitor/actions
2. Report error to user
3. Offer to delete tag and retry:
   ```powershell
   git tag -d v[VERSION]
   git push origin :refs/tags/v[VERSION]
   ```

### **If Tag Already Exists:**
1. List existing tags
2. Suggest incrementing version or deleting old tag

### **If PR Creation Fails:**
1. Report error details
2. Provide manual PR creation instructions

---

## Summary of Automation

### **Fully Automated:**
- ✅ Feature branch creation
- ✅ File updates (version bumping)
- ✅ Git commits and pushes
- ✅ PR creation via GitHub MCP
- ✅ Tag creation and push
- ✅ Release verification
- ✅ Branch cleanup

### **Human Approval Gates:**
1. Before updating main branch
2. Before creating feature branch
3. Before pushing version changes
4. After PR created (manual review and merge required)
5. Before pushing release tag
6. After workflow triggered (wait for build completion)
7. Before cleaning up branches

### **Manual Steps:**
- PR review and merge (intentionally manual for quality control)
- Confirm workflow completion (watch build progress)

---

## GitHub MCP Tools Used

- `mcp_github_github_get_me` - Verify authentication
- `mcp_github_github_create_pull_request` - Create version bump PR
- `mcp_github_github_list_tags` - Check existing tags
- `mcp_github_github_get_release_by_tag` - Verify release creation
- Standard file operations (`read_file`, `replace_string_in_file`)
- Terminal commands for git operations

---

## Usage Example

**User says:** "I want to release version 0.3.0 with the new analytics dashboard"

**Agent responds:**
1. ✅ Creates `release/v0.3.0` branch
2. ✅ Updates version files
3. ✅ Creates PR with description
4. ⏸️ Waits for user to merge PR
5. ✅ Creates and pushes tag `v0.3.0`
6. ✅ Monitors release workflow
7. ✅ Verifies release and reports success
8. ✅ Cleans up feature branch

**Total interaction time:** ~5 minutes (including ~3 min build wait)
**Manual steps:** PR review/merge only
