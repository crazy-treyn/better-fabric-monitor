# Notebook Deep Link Regression

## Summary
- Users who upgraded from builds prior to `notebook_sessions` support keep notebook job links that point at the legacy `jobId` path.
- Subsequent refreshes only enrich freshly fetched jobs; cached jobs retain the stale URL even after Livy sessions are synced.
- Environments with paused Fabric capacities exacerbate the problem because the Livy session API cannot return data, so no backfill occurs until the capacity resumes.

## User Impact
- Notebook executions that predate the upgrade never transition to the Livy-based URL. The link typically opens the Fabric Spark monitor landing page instead of the specific run.
- Mixed datasets (some new, some historical) show inconsistent behavior, which is confusing to users and erodes trust.
- When capacities are paused, _all_ notebook runs during that window fall back to `jobId` indefinitely because we never retry URL generation once the capacity is resumed.

## Root Cause Analysis
1. **Schema migration sequence**
   - Cached notebook jobs were written before we persisted Livy sessions, so their URLs were already stored with `jobId`.
2. **GetJobs flow**
   - `GetJobs()` loads cached rows (`GetJobsFromCache()`) _before_ the Livy sync runs.
   - We update URLs only for the freshly fetched jobs using the Livy IDs retrieved in the same call.
   - Cached rows are appended unchanged, so their `fabricUrl` never benefits from the new data.
3. **Livy backfill timing**
   - If `GetRecentJobs` returns zero rows (common right after an upgrade), the Livy sync short-circuits because we guard it with `len(jobs) > 0`.
   - Paused capacities make the Livy API respond with errors; we log the warning but the affected jobs are not queued for retry.

## Recommendations
1. **Regenerate cached URLs after Livy sync**
   - After populating `livyIDMap`, iterate the cached set before merging so their `fabricUrl` uses the freshest Livy data.
   - Consider adding a small helper that re-invokes `GenerateFabricURL` for any job where `fabricUrl` is empty or still contains the fallback pattern.
2. **Always run notebook session sync when Livy data might be stale**
   - Remove the `len(jobs) > 0` guard; if `startTimeFrom != nil`, run `SyncNotebookSessions()` even without fresh jobs.
   - Trigger a sync during startup or on demand (e.g., first dashboard load) to upgrade legacy caches automatically.
3. **Retry strategy for paused capacities**
   - Detect Livy API failures that reference paused capacities (HTTP `409/423` or specific error codes) and store the affected `job_instance_id` values in a retry queue.
   - On the next successful refresh, reprocess the queue before returning data to the UI.
4. **Diagnostics**
   - Surface a warning in the UI when we fall back to `jobId` so users know the link may fail and to retry after the next refresh.

## Next Steps
- Implement the enrichment changes inside `GetJobs` and validate that historical notebook rows pick up the Livy URLs after a single refresh.
- Build the retry/queue mechanism so paused capacities are handled gracefully and without manual intervention.
- Add regression tests around cached job URL regeneration, including simulated Livy API failure scenarios.
