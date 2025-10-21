<script>
    import { onMount } from "svelte";
    import { authStore, authActions } from "../stores/auth.js";
    import { filterStore } from "../stores/filters.js";
    import Analytics from "./Analytics.svelte";

    let workspaces = [];
    let jobs = [];
    let isLoading = true;
    let sidebarWidth = 350; // Default width in pixels (~20% wider than 256px)
    let isResizing = false;
    let currentView = "jobs"; // 'jobs' or 'analytics'

    // Filter states
    let filterJob = "";
    let filterType = "";
    let filterStatus = "";
    let workspaceSearchText = "";
    let hasLoadedData = false;
    let lastSyncTime = "";

    // Subscribe to filter store for workspace selection
    let selectedWorkspaceIds = new Set();
    filterStore.subscribe((state) => {
        selectedWorkspaceIds = state.selectedWorkspaceIds;
    });

    onMount(async () => {
        // Load cached data from DuckDB on mount
        await loadCachedData();
    });

    async function loadCachedData() {
        try {
            isLoading = true;
            // Load from local cache (DuckDB)
            const cachedWorkspaces =
                (await window.go.main.App.GetWorkspacesFromCache()) || [];
            if (cachedWorkspaces.length > 0) {
                workspaces = cachedWorkspaces;
                console.log(
                    `Loaded ${cachedWorkspaces.length} workspaces from cache`,
                );
            }

            const cachedJobs =
                (await window.go.main.App.GetJobsFromCache()) || [];
            if (cachedJobs.length > 0) {
                jobs = cachedJobs;
                hasLoadedData = true;
                console.log(`Loaded ${cachedJobs.length} jobs from cache`);
            }

            // Get last sync time
            lastSyncTime = (await window.go.main.App.GetLastSyncTime()) || "";
        } catch (error) {
            console.error("Failed to load cached data:", error);
        } finally {
            isLoading = false;
        }
    }

    async function loadData() {
        try {
            isLoading = true;
            // Load fresh data from Fabric API (also persists to cache)
            workspaces = (await window.go.main.App.GetWorkspaces()) || [];
            jobs = (await window.go.main.App.GetJobs()) || [];
            hasLoadedData = true;

            // Update last sync time
            lastSyncTime = new Date().toISOString();
        } catch (error) {
            console.error("Failed to load data:", error);
        } finally {
            isLoading = false;
        }
    }

    async function handleLogout() {
        await authActions.logout();
    }

    function formatDate(dateString) {
        if (!dateString) return "N/A";
        return new Date(dateString).toLocaleString();
    }

    function formatDuration(durationMs) {
        if (!durationMs || durationMs < 0) return "N/A";

        const seconds = Math.floor(durationMs / 1000);

        if (seconds < 60) {
            return `${seconds}s`;
        }

        const minutes = Math.floor(seconds / 60);
        if (minutes < 60) {
            const remainingSeconds = seconds % 60;
            return remainingSeconds === 0
                ? `${minutes}m`
                : `${minutes}m ${remainingSeconds}s`;
        }

        const hours = Math.floor(minutes / 60);
        const remainingMinutes = minutes % 60;
        return remainingMinutes === 0
            ? `${hours}h`
            : `${hours}h ${remainingMinutes}m`;
    }

    function getStatusColor(status) {
        switch (status?.toLowerCase()) {
            case "completed":
                return "text-green-400";
            case "failed":
                return "text-red-400";
            case "running":
                return "text-blue-400";
            default:
                return "text-slate-400";
        }
    }

    function startResize(e) {
        isResizing = true;
        e.preventDefault();
    }

    function handleMouseMove(e) {
        if (isResizing) {
            const newWidth = e.clientX;
            if (newWidth >= 200 && newWidth <= 500) {
                sidebarWidth = newWidth;
            }
        }
    }

    function stopResize() {
        isResizing = false;
    }

    // Handle workspace selection
    function toggleWorkspaceSelection(workspaceId, event) {
        // Support Ctrl/Shift+Click for quick multi-select
        if (event?.ctrlKey || event?.shiftKey) {
            filterStore.toggleWorkspace(workspaceId);
        } else {
            filterStore.toggleWorkspace(workspaceId);
        }
    }

    // Select all visible workspaces (based on search)
    function selectAllWorkspaces() {
        filterStore.selectAllWorkspaces(filteredWorkspaces);
    }

    // Clear all workspace selections
    function clearAllWorkspaces() {
        filterStore.clearWorkspaces();
    }

    // Computed filtered jobs
    $: filteredJobs = jobs.filter((job) => {
        const matchesJob =
            !filterJob ||
            (job.itemDisplayName || "")
                .toLowerCase()
                .includes(filterJob.toLowerCase());
        const matchesType = !filterType || job.itemType === filterType;
        const matchesStatus = !filterStatus || job.status === filterStatus;
        const matchesWorkspace =
            selectedWorkspaceIds.size === 0 ||
            selectedWorkspaceIds.has(job.workspaceId);
        return matchesJob && matchesType && matchesStatus && matchesWorkspace;
    });

    // Computed filtered workspaces based on search text
    $: filteredWorkspaces = workspaces.filter((ws) =>
        (ws.displayName || ws.id)
            .toLowerCase()
            .includes(workspaceSearchText.toLowerCase()),
    );

    // Get unique values for filters
    $: uniqueTypes = [
        ...new Set(jobs.map((j) => j.itemType).filter(Boolean)),
    ].sort();
    $: uniqueStatuses = [
        ...new Set(jobs.map((j) => j.status).filter(Boolean)),
    ].sort();
</script>

<svelte:window on:mousemove={handleMouseMove} on:mouseup={stopResize} />

<div class="h-screen flex flex-col bg-slate-900">
    <!-- Header -->
    <header class="bg-slate-800 border-b border-slate-700 px-6 py-4">
        <div class="flex items-center justify-between">
            <div class="flex items-center space-x-4">
                <h1 class="text-xl font-semibold text-white">Fabric Monitor</h1>
                <span class="text-sm text-slate-400"
                    >Tenant: {$authStore.tenantId}</span
                >
                <div class="flex gap-2 ml-6">
                    <button
                        on:click={() => (currentView = "jobs")}
                        class="px-4 py-2 text-sm rounded-md transition-colors {currentView ===
                        'jobs'
                            ? 'bg-primary-600 text-white'
                            : 'text-slate-300 hover:text-white hover:bg-slate-700'}"
                    >
                        Jobs
                    </button>
                    <button
                        on:click={() => (currentView = "analytics")}
                        class="px-4 py-2 text-sm rounded-md transition-colors {currentView ===
                        'analytics'
                            ? 'bg-primary-600 text-white'
                            : 'text-slate-300 hover:text-white hover:bg-slate-700'}"
                    >
                        Analytics
                    </button>
                </div>
            </div>
            <div class="flex items-center gap-3">
                {#if hasLoadedData && currentView === "jobs"}
                    <div class="text-sm text-slate-400">
                        {#if lastSyncTime}
                            Last synced: {new Date(
                                lastSyncTime,
                            ).toLocaleString()}
                        {/if}
                    </div>
                    <button
                        on:click={loadData}
                        class="px-4 py-2 text-sm bg-primary-600 hover:bg-primary-700 text-white rounded-md transition-colors"
                        disabled={isLoading}
                    >
                        {isLoading ? "Loading..." : "Refresh from API"}
                    </button>
                {/if}
                <button
                    on:click={handleLogout}
                    class="px-4 py-2 text-sm text-slate-300 hover:text-white hover:bg-slate-700 rounded-md transition-colors"
                >
                    Sign Out
                </button>
            </div>
        </div>
    </header>

    <!-- Main Content -->
    <main class="flex-1 overflow-hidden">
        {#if currentView === "analytics"}
            <Analytics />
        {:else if !hasLoadedData && !isLoading}
            <div class="flex items-center justify-center h-full">
                <div class="text-center">
                    <svg
                        class="mx-auto h-16 w-16 text-slate-400 mb-4"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                    >
                        <path
                            stroke-linecap="round"
                            stroke-linejoin="round"
                            stroke-width="2"
                            d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"
                        />
                    </svg>
                    <h2 class="text-xl font-semibold text-white mb-2">
                        No cached data found
                    </h2>
                    <p class="text-slate-400 mb-6">
                        Click the button below to fetch workspaces and job
                        instances from the Fabric API
                    </p>
                    <button
                        on:click={loadData}
                        class="px-6 py-3 bg-primary-600 hover:bg-primary-700 text-white font-medium rounded-lg transition-colors"
                    >
                        Load Data from API
                    </button>
                </div>
            </div>
        {:else if isLoading}
            <div class="flex items-center justify-center h-full">
                <div class="text-center">
                    <div
                        class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-500 mx-auto mb-4"
                    ></div>
                    <p class="text-slate-400">Loading your Fabric data...</p>
                </div>
            </div>
        {:else}
            <div class="h-full flex">
                <!-- Sidebar -->
                <div
                    class="bg-slate-800 border-r border-slate-700 flex flex-col overflow-hidden"
                    style="width: {sidebarWidth}px; min-width: 200px; max-width: 500px;"
                >
                    <div class="p-4">
                        <div class="flex items-center justify-between mb-2">
                            <h2 class="text-lg font-semibold text-white">
                                Workspaces
                                {#if selectedWorkspaceIds.size > 0}
                                    <span
                                        class="ml-2 text-sm font-normal text-primary-400"
                                    >
                                        ({selectedWorkspaceIds.size} selected)
                                    </span>
                                {/if}
                            </h2>
                        </div>

                        <!-- Search box -->
                        <input
                            type="text"
                            bind:value={workspaceSearchText}
                            placeholder="Search workspaces..."
                            class="w-full px-3 py-2 mb-2 bg-slate-700 border border-slate-600 rounded-md text-white text-sm placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-primary-500"
                        />

                        <!-- Selection controls -->
                        <div class="flex gap-2">
                            <button
                                on:click={selectAllWorkspaces}
                                class="flex-1 px-2 py-1 text-xs bg-slate-700 hover:bg-slate-600 text-slate-300 rounded transition-colors"
                            >
                                Select All
                            </button>
                            <button
                                on:click={clearAllWorkspaces}
                                class="flex-1 px-2 py-1 text-xs bg-slate-700 hover:bg-slate-600 text-slate-300 rounded transition-colors"
                            >
                                Clear All
                            </button>
                        </div>
                    </div>

                    <div class="space-y-2 overflow-y-auto flex-1 px-4 pb-4">
                        {#each filteredWorkspaces as workspace}
                            <div
                                class="p-3 bg-slate-700 rounded-lg hover:bg-slate-600 transition-colors {selectedWorkspaceIds.has(
                                    workspace.id,
                                )
                                    ? 'ring-2 ring-primary-500 bg-slate-600'
                                    : ''}"
                                title={workspace.displayName || workspace.id}
                            >
                                <label
                                    class="flex items-center gap-2 cursor-pointer"
                                >
                                    <input
                                        type="checkbox"
                                        checked={selectedWorkspaceIds.has(
                                            workspace.id,
                                        )}
                                        on:change={(e) =>
                                            toggleWorkspaceSelection(
                                                workspace.id,
                                                e,
                                            )}
                                        class="h-4 w-4 rounded border-slate-500 bg-slate-600 text-primary-600 focus:ring-2 focus:ring-primary-500 focus:ring-offset-0 flex-shrink-0"
                                    />
                                    <div class="flex-1 min-w-0">
                                        <h3
                                            class="text-sm font-medium text-white truncate"
                                        >
                                            {workspace.displayName ||
                                                workspace.id}
                                        </h3>
                                    </div>
                                </label>
                            </div>
                        {/each}
                        {#if filteredWorkspaces.length === 0}
                            <p class="text-slate-400 text-sm">
                                {workspaces.length === 0
                                    ? "No workspaces found"
                                    : "No matching workspaces"}
                            </p>
                        {/if}
                    </div>
                </div>

                <!-- Resize Handle -->
                <div
                    class="w-1 bg-slate-700 hover:bg-primary-500 cursor-col-resize transition-colors"
                    on:mousedown={startResize}
                    role="separator"
                    aria-label="Resize sidebar"
                    tabindex="0"
                ></div>

                <!-- Main Panel -->
                <div class="flex-1 p-6 overflow-auto">
                    <div class="mb-6">
                        <h2 class="text-2xl font-bold text-white mb-4">
                            Recent Jobs
                        </h2>

                        <!-- Filters -->
                        <div class="mb-4 flex gap-4">
                            <div class="flex-1">
                                <label
                                    class="block text-sm font-medium text-slate-300 mb-1"
                                    >Search Job Name</label
                                >
                                <input
                                    type="text"
                                    bind:value={filterJob}
                                    placeholder="Filter by job name..."
                                    class="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-primary-500"
                                />
                            </div>
                            <div class="w-48">
                                <label
                                    class="block text-sm font-medium text-slate-300 mb-1"
                                    >Type</label
                                >
                                <select
                                    bind:value={filterType}
                                    class="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
                                >
                                    <option value="">All Types</option>
                                    {#each uniqueTypes as type}
                                        <option value={type}>{type}</option>
                                    {/each}
                                </select>
                            </div>
                            <div class="w-48">
                                <label
                                    class="block text-sm font-medium text-slate-300 mb-1"
                                    >Status</label
                                >
                                <select
                                    bind:value={filterStatus}
                                    class="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
                                >
                                    <option value="">All Statuses</option>
                                    {#each uniqueStatuses as status}
                                        <option value={status}>{status}</option>
                                    {/each}
                                </select>
                            </div>
                        </div>

                        {#if filteredJobs.length > 0}
                            <div
                                class="bg-slate-800 rounded-lg overflow-hidden"
                            >
                                <table class="w-full">
                                    <thead class="bg-slate-700">
                                        <tr>
                                            <th
                                                class="px-4 py-3 text-left text-xs font-medium text-slate-300 uppercase tracking-wider"
                                                >Job</th
                                            >
                                            <th
                                                class="px-4 py-3 text-left text-xs font-medium text-slate-300 uppercase tracking-wider"
                                                >Type</th
                                            >
                                            <th
                                                class="px-4 py-3 text-left text-xs font-medium text-slate-300 uppercase tracking-wider"
                                                >Status</th
                                            >
                                            <th
                                                class="px-4 py-3 text-left text-xs font-medium text-slate-300 uppercase tracking-wider"
                                                >Started</th
                                            >
                                            <th
                                                class="px-4 py-3 text-left text-xs font-medium text-slate-300 uppercase tracking-wider"
                                                >Duration</th
                                            >
                                        </tr>
                                    </thead>
                                    <tbody class="divide-y divide-slate-700">
                                        {#each filteredJobs as job}
                                            <tr class="hover:bg-slate-700/50">
                                                <td class="px-4 py-3">
                                                    <div
                                                        class="text-sm text-white"
                                                    >
                                                        {job.itemDisplayName ||
                                                            job.itemId}
                                                    </div>
                                                    <div
                                                        class="text-xs text-slate-400"
                                                    >
                                                        {job.workspaceName ||
                                                            job.workspaceId}
                                                    </div>
                                                </td>
                                                <td class="px-4 py-3">
                                                    <div
                                                        class="text-sm text-slate-300"
                                                    >
                                                        {job.itemType || "N/A"}
                                                    </div>
                                                    <div
                                                        class="text-xs text-slate-400"
                                                    >
                                                        {job.jobType}
                                                    </div>
                                                </td>
                                                <td class="px-4 py-3">
                                                    <span
                                                        class="inline-flex px-2 py-1 text-xs font-semibold rounded-full {getStatusColor(
                                                            job.status,
                                                        )} bg-slate-700"
                                                    >
                                                        {job.status ||
                                                            "Unknown"}
                                                    </span>
                                                </td>
                                                <td
                                                    class="px-4 py-3 text-sm text-slate-300"
                                                >
                                                    {formatDate(job.startTime)}
                                                </td>
                                                <td
                                                    class="px-4 py-3 text-sm text-slate-300"
                                                >
                                                    {formatDuration(
                                                        job.durationMs,
                                                    )}
                                                </td>
                                            </tr>
                                        {/each}
                                    </tbody>
                                </table>
                                <div
                                    class="px-4 py-3 bg-slate-700/50 text-sm text-slate-400"
                                >
                                    Showing {filteredJobs.length} of {jobs.length}
                                    jobs
                                </div>
                            </div>
                        {:else}
                            <div
                                class="bg-slate-800 rounded-lg p-8 text-center"
                            >
                                <svg
                                    class="mx-auto h-12 w-12 text-slate-400 mb-4"
                                    fill="none"
                                    viewBox="0 0 24 24"
                                    stroke="currentColor"
                                >
                                    <path
                                        stroke-linecap="round"
                                        stroke-linejoin="round"
                                        stroke-width="2"
                                        d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
                                    />
                                </svg>
                                <h3 class="text-lg font-medium text-white mb-2">
                                    {jobs.length > 0
                                        ? "No matching jobs"
                                        : "No jobs found"}
                                </h3>
                                <p class="text-slate-400">
                                    {jobs.length > 0
                                        ? "Try adjusting your filters"
                                        : "Jobs will appear here once they start running in your workspaces."}
                                </p>
                            </div>
                        {/if}
                    </div>
                </div>
            </div>
        {/if}
    </main>
</div>
