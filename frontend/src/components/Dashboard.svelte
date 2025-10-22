<script>
    import { onMount } from "svelte";
    import { authStore, authActions } from "../stores/auth.js";
    import { filterStore } from "../stores/filters.js";
    import Analytics from "./Analytics.svelte";
    import Logs from "./Logs.svelte";

    let workspaces = [];
    let jobs = [];
    let isLoading = true;
    let sidebarWidth = 350; // Default width in pixels (~20% wider than 256px)
    let isResizing = false;
    let currentView = "jobs"; // 'jobs', 'analytics', or 'logs'

    // Filter states
    let filterJob = "";
    let filterType = "";
    let filterStatus = "";
    let workspaceSearchText = "";
    let hasLoadedData = false;
    let lastSyncTime = "";

    // Expanded job state for hierarchical view
    let expandedJobs = new Set();
    let jobChildrenCache = new Map(); // Cache child executions per job
    let loadingChildren = new Set(); // Track which jobs are loading children

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

    // Toggle expansion of a job to show/hide children
    async function toggleJobExpansion(jobId) {
        if (expandedJobs.has(jobId)) {
            expandedJobs.delete(jobId);
            expandedJobs = expandedJobs; // Trigger reactivity
        } else {
            expandedJobs.add(jobId);
            expandedJobs = expandedJobs; // Trigger reactivity

            // Load children if not already cached
            if (!jobChildrenCache.has(jobId)) {
                await loadChildExecutions(jobId);
            }
        }
    }

    // Load child executions for a job
    async function loadChildExecutions(jobId) {
        loadingChildren.add(jobId);
        loadingChildren = loadingChildren;

        try {
            const result = await window.go.main.App.GetChildExecutions(jobId);
            if (result.error) {
                console.error(
                    `Failed to load children for job ${jobId}:`,
                    result.error,
                );
                jobChildrenCache.set(jobId, []);
            } else {
                jobChildrenCache.set(jobId, result.children || []);
            }
            jobChildrenCache = jobChildrenCache; // Trigger reactivity
        } catch (error) {
            console.error(`Failed to load children for job ${jobId}:`, error);
            jobChildrenCache.set(jobId, []);
        } finally {
            loadingChildren.delete(jobId);
            loadingChildren = loadingChildren;
        }
    }

    // Get icon for activity type
    function getActivityIcon(activityType) {
        switch (activityType?.toLowerCase()) {
            case "executepipeline":
                return "🔗"; // Pipeline
            case "tridentnotebook":
                return "📓"; // Notebook (Trident internal type)
            case "executenotebook":
            case "dataflownotebook":
                return "📓"; // Notebook
            case "copy":
                return "📋";
            case "foreach":
                return "🔄";
            case "script":
                return "📝";
            case "wait":
                return "⏱️";
            default:
                return "▪️";
        }
    }

    // Get display name for activity type (maps internal names to friendly names)
    function getActivityDisplayName(activityType) {
        if (activityType?.toLowerCase() === "tridentnotebook") {
            return "Notebook";
        }
        return activityType;
    }

    // Recursively load all children for a job (for nested hierarchies)
    async function loadAllChildren(jobId, depth = 0, visited = new Set()) {
        // Prevent infinite loops
        if (visited.has(jobId) || depth > 10) return;
        visited.add(jobId);

        if (!jobChildrenCache.has(jobId)) {
            await loadChildExecutions(jobId);
        }

        const children = jobChildrenCache.get(jobId) || [];
        for (const child of children) {
            if (child.childJobInstanceId) {
                await loadAllChildren(
                    child.childJobInstanceId,
                    depth + 1,
                    visited,
                );
            }
        }
    }
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
                    <button
                        on:click={() => (currentView = "logs")}
                        class="px-4 py-2 text-sm rounded-md transition-colors {currentView ===
                        'logs'
                            ? 'bg-primary-600 text-white'
                            : 'text-slate-300 hover:text-white hover:bg-slate-700'}"
                    >
                        Logs
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
        {:else if currentView === "logs"}
            <Logs />
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

                        <!-- Expansion Controls for DataPipelines -->
                        {#if filteredJobs.some((j) => j.itemType === "DataPipeline")}
                            <div class="mb-4 flex gap-2">
                                <button
                                    on:click={async () => {
                                        const pipelineJobs =
                                            filteredJobs.filter(
                                                (j) =>
                                                    j.itemType ===
                                                    "DataPipeline",
                                            );
                                        for (const job of pipelineJobs) {
                                            if (!expandedJobs.has(job.id)) {
                                                expandedJobs.add(job.id);
                                                if (
                                                    !jobChildrenCache.has(
                                                        job.id,
                                                    )
                                                ) {
                                                    await loadChildExecutions(
                                                        job.id,
                                                    );
                                                }
                                            }
                                        }
                                        expandedJobs = expandedJobs;
                                    }}
                                    class="px-3 py-1.5 text-xs bg-slate-700 hover:bg-slate-600 text-slate-300 rounded transition-colors"
                                >
                                    📂 Expand All Pipelines
                                </button>
                                <button
                                    on:click={() => {
                                        expandedJobs.clear();
                                        expandedJobs = expandedJobs;
                                    }}
                                    class="px-3 py-1.5 text-xs bg-slate-700 hover:bg-slate-600 text-slate-300 rounded transition-colors"
                                >
                                    📁 Collapse All
                                </button>
                            </div>
                        {/if}

                        {#if filteredJobs.length > 0}
                            <div
                                class="bg-slate-800 rounded-lg overflow-hidden"
                            >
                                <table class="w-full">
                                    <thead class="bg-slate-700">
                                        <tr>
                                            <th
                                                class="px-4 py-3 text-left text-xs font-medium text-slate-300 uppercase tracking-wider w-10"
                                            ></th>
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
                                            <!-- Parent Job Row -->
                                            <tr class="hover:bg-slate-700/50">
                                                <td class="px-4 py-3">
                                                    {#if job.itemType === "DataPipeline"}
                                                        <button
                                                            on:click={() =>
                                                                toggleJobExpansion(
                                                                    job.id,
                                                                )}
                                                            class="text-slate-400 hover:text-white transition-colors"
                                                            title="Show/hide child executions"
                                                        >
                                                            {#if loadingChildren.has(job.id)}
                                                                <svg
                                                                    class="animate-spin h-4 w-4"
                                                                    xmlns="http://www.w3.org/2000/svg"
                                                                    fill="none"
                                                                    viewBox="0 0 24 24"
                                                                >
                                                                    <circle
                                                                        class="opacity-25"
                                                                        cx="12"
                                                                        cy="12"
                                                                        r="10"
                                                                        stroke="currentColor"
                                                                        stroke-width="4"
                                                                    ></circle>
                                                                    <path
                                                                        class="opacity-75"
                                                                        fill="currentColor"
                                                                        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                                                                    ></path>
                                                                </svg>
                                                            {:else if expandedJobs.has(job.id)}
                                                                <svg
                                                                    class="h-4 w-4"
                                                                    fill="none"
                                                                    stroke="currentColor"
                                                                    viewBox="0 0 24 24"
                                                                >
                                                                    <path
                                                                        stroke-linecap="round"
                                                                        stroke-linejoin="round"
                                                                        stroke-width="2"
                                                                        d="M19 9l-7 7-7-7"
                                                                    />
                                                                </svg>
                                                            {:else}
                                                                <svg
                                                                    class="h-4 w-4"
                                                                    fill="none"
                                                                    stroke="currentColor"
                                                                    viewBox="0 0 24 24"
                                                                >
                                                                    <path
                                                                        stroke-linecap="round"
                                                                        stroke-linejoin="round"
                                                                        stroke-width="2"
                                                                        d="M9 5l7 7-7 7"
                                                                    />
                                                                </svg>
                                                            {/if}
                                                        </button>
                                                    {/if}
                                                </td>
                                                <td class="px-4 py-3">
                                                    <div
                                                        class="text-sm text-white font-medium flex items-center gap-2"
                                                    >
                                                        <span>
                                                            {job.itemDisplayName ||
                                                                job.itemId}
                                                        </span>
                                                        {#if job.itemType === "DataPipeline" && jobChildrenCache.has(job.id)}
                                                            {@const childCount =
                                                                jobChildrenCache.get(
                                                                    job.id,
                                                                )?.length || 0}
                                                            {#if childCount > 0}
                                                                <span
                                                                    class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-primary-900/50 text-primary-300 border border-primary-700"
                                                                    title="{childCount} child execution{childCount ===
                                                                    1
                                                                        ? ''
                                                                        : 's'}"
                                                                >
                                                                    {childCount}
                                                                </span>
                                                            {/if}
                                                        {/if}
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

                                            <!-- Child Executions (Nested) -->
                                            {#if expandedJobs.has(job.id) && jobChildrenCache.has(job.id)}
                                                {#each jobChildrenCache.get(job.id) || [] as child, idx}
                                                    <tr
                                                        class="bg-slate-800/50 hover:bg-slate-700/30"
                                                    >
                                                        <td
                                                            class="px-4 py-2 text-right"
                                                        >
                                                            {#if child.childJobInstanceId && child.activityType === "ExecutePipeline"}
                                                                <button
                                                                    on:click={() =>
                                                                        toggleJobExpansion(
                                                                            child.childJobInstanceId,
                                                                        )}
                                                                    class="text-slate-500 hover:text-slate-300 transition-colors"
                                                                    title="Show/hide nested executions"
                                                                >
                                                                    {#if loadingChildren.has(child.childJobInstanceId)}
                                                                        <svg
                                                                            class="animate-spin h-3 w-3"
                                                                            xmlns="http://www.w3.org/2000/svg"
                                                                            fill="none"
                                                                            viewBox="0 0 24 24"
                                                                        >
                                                                            <circle
                                                                                class="opacity-25"
                                                                                cx="12"
                                                                                cy="12"
                                                                                r="10"
                                                                                stroke="currentColor"
                                                                                stroke-width="4"

                                                                            ></circle>
                                                                            <path
                                                                                class="opacity-75"
                                                                                fill="currentColor"
                                                                                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"

                                                                            ></path>
                                                                        </svg>
                                                                    {:else if expandedJobs.has(child.childJobInstanceId)}
                                                                        <span
                                                                            class="text-xs"
                                                                            >└▼</span
                                                                        >
                                                                    {:else}
                                                                        <span
                                                                            class="text-xs"
                                                                            >└▶</span
                                                                        >
                                                                    {/if}
                                                                </button>
                                                            {:else}
                                                                <span
                                                                    class="text-slate-500 text-xs"
                                                                    >└─</span
                                                                >
                                                            {/if}
                                                        </td>
                                                        <td
                                                            class="px-4 py-2 pl-8"
                                                        >
                                                            <div
                                                                class="text-sm text-slate-300 flex items-center gap-2"
                                                            >
                                                                <span
                                                                    class="text-base"
                                                                    >{getActivityIcon(
                                                                        child.activityType,
                                                                    )}</span
                                                                >
                                                                <span>
                                                                    {child.activityName}
                                                                </span>
                                                            </div>
                                                            {#if child.childPipelineName || child.childNotebookName}
                                                                <div
                                                                    class="text-xs text-slate-400 ml-7"
                                                                >
                                                                    {child.childPipelineName ||
                                                                        child.childNotebookName}
                                                                </div>
                                                            {/if}
                                                            {#if child.error}
                                                                <div
                                                                    class="text-xs text-red-400 ml-7 mt-1"
                                                                >
                                                                    ⚠️ {child.error}
                                                                </div>
                                                            {/if}
                                                        </td>
                                                        <td class="px-4 py-2">
                                                            <div
                                                                class="text-xs text-slate-400"
                                                            >
                                                                {getActivityDisplayName(
                                                                    child.activityType,
                                                                )}
                                                            </div>
                                                        </td>
                                                        <td class="px-4 py-2">
                                                            <span
                                                                class="inline-flex px-2 py-0.5 text-xs font-semibold rounded-full {getStatusColor(
                                                                    child.status,
                                                                )} bg-slate-700/50"
                                                            >
                                                                {child.status}
                                                            </span>
                                                        </td>
                                                        <td
                                                            class="px-4 py-2 text-xs text-slate-400"
                                                        >
                                                            {formatDate(
                                                                child.activityRunStart,
                                                            )}
                                                        </td>
                                                        <td
                                                            class="px-4 py-2 text-xs text-slate-400"
                                                        >
                                                            {formatDuration(
                                                                child.durationMs,
                                                            )}
                                                        </td>
                                                    </tr>

                                                    <!-- Nested Children (Recursive - if this child is a pipeline with its own children) -->
                                                    {#if child.childJobInstanceId && expandedJobs.has(child.childJobInstanceId) && jobChildrenCache.has(child.childJobInstanceId)}
                                                        {#each jobChildrenCache.get(child.childJobInstanceId) || [] as grandchild}
                                                            <tr
                                                                class="bg-slate-800/30 hover:bg-slate-700/20"
                                                            >
                                                                <td
                                                                    class="px-4 py-2 text-right"
                                                                >
                                                                    <span
                                                                        class="text-slate-600 text-xs"
                                                                        >└─└─</span
                                                                    >
                                                                </td>
                                                                <td
                                                                    class="px-4 py-2 pl-16"
                                                                >
                                                                    <div
                                                                        class="text-sm text-slate-400 flex items-center gap-2"
                                                                    >
                                                                        <span
                                                                            class="text-base"
                                                                            >{getActivityIcon(
                                                                                grandchild.activityType,
                                                                            )}</span
                                                                        >
                                                                        <span>
                                                                            {grandchild.activityName}
                                                                        </span>
                                                                    </div>
                                                                    {#if grandchild.childPipelineName || grandchild.childNotebookName}
                                                                        <div
                                                                            class="text-xs text-slate-500 ml-7"
                                                                        >
                                                                            {grandchild.childPipelineName ||
                                                                                grandchild.childNotebookName}
                                                                        </div>
                                                                    {/if}
                                                                </td>
                                                                <td
                                                                    class="px-4 py-2"
                                                                >
                                                                    <div
                                                                        class="text-xs text-slate-500"
                                                                    >
                                                                        {getActivityDisplayName(
                                                                            grandchild.activityType,
                                                                        )}
                                                                    </div>
                                                                </td>
                                                                <td
                                                                    class="px-4 py-2"
                                                                >
                                                                    <span
                                                                        class="inline-flex px-2 py-0.5 text-xs rounded-full {getStatusColor(
                                                                            grandchild.status,
                                                                        )} bg-slate-700/30"
                                                                    >
                                                                        {grandchild.status}
                                                                    </span>
                                                                </td>
                                                                <td
                                                                    class="px-4 py-2 text-xs text-slate-500"
                                                                >
                                                                    {formatDate(
                                                                        grandchild.activityRunStart,
                                                                    )}
                                                                </td>
                                                                <td
                                                                    class="px-4 py-2 text-xs text-slate-500"
                                                                >
                                                                    {formatDuration(
                                                                        grandchild.durationMs,
                                                                    )}
                                                                </td>
                                                            </tr>
                                                        {/each}
                                                    {/if}
                                                {/each}
                                            {/if}
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
