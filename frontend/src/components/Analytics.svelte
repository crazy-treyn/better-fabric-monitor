<script>
    import { onMount, afterUpdate } from "svelte";
    import { filterStore } from "../stores/filters.js";
    import Chart from "chart.js/auto";
    import FabricLink from "./FabricLink.svelte";

    let analytics = null;
    let isLoading = true;
    let selectedDays = 7;
    let error = null;
    let chartCanvas;
    let chartInstance = null;
    let workspaceChartCanvas;
    let workspaceChartInstance = null;
    let jobTypeChartCanvas;
    let jobTypeChartInstance = null;
    let selectedWorkspace = null;
    let selectedJobType = null;
    let selectedDate = null;
    let drillDownData = null;

    // Filter states
    let selectedWorkspaceIds = new Set();
    let selectedItemTypes = new Set();
    let itemNameSearch = "";
    let availableItemTypes = [];
    let allWorkspaces = [];
    let workspaceSearchText = "";
    let itemTypeSearchText = "";
    let showWorkspaceDropdown = false;
    let showItemTypeDropdown = false;
    let isInitialized = false;

    // Subscribe to filter store for workspace selection sync
    filterStore.subscribe((state) => {
        selectedWorkspaceIds = state.selectedWorkspaceIds;
        // Only auto-refresh after initial load is complete
        if (isInitialized && analytics !== null) {
            loadAvailableItemTypes();
            loadAnalytics();
        }
    });

    onMount(async () => {
        await loadAnalytics();
    });

    afterUpdate(() => {
        // Update chart when analytics data changes
        if (
            chartCanvas &&
            analytics?.dailyStats &&
            analytics.dailyStats.length > 0
        ) {
            updateChart();
        }
        if (
            workspaceChartCanvas &&
            analytics?.workspaceStats &&
            analytics.workspaceStats.length > 0
        ) {
            updateWorkspaceChart();
        }
        if (
            jobTypeChartCanvas &&
            analytics?.itemTypeStats &&
            analytics.itemTypeStats.length > 0
        ) {
            updateJobTypeChart();
        }
    });

    function updateChart() {
        // Destroy existing chart if it exists
        if (chartInstance) {
            chartInstance.destroy();
        }

        if (
            !chartCanvas ||
            !analytics?.dailyStats ||
            analytics.dailyStats.length === 0
        ) {
            return;
        }

        const ctx = chartCanvas.getContext("2d");

        // Prepare data - reverse to show chronological order (oldest to newest)
        const labels = analytics.dailyStats.map((stat) => {
            const date = new Date(stat.date);
            return date.toLocaleDateString("en-US", {
                month: "short",
                day: "numeric",
            });
        });

        const successfulData = analytics.dailyStats.map(
            (stat) => stat.successful,
        );
        const failedData = analytics.dailyStats.map((stat) => stat.failed);

        chartInstance = new Chart(ctx, {
            type: "line",
            data: {
                labels: labels,
                datasets: [
                    {
                        label: "Successful",
                        data: successfulData,
                        borderColor: "rgb(74, 222, 128)",
                        backgroundColor: "rgba(74, 222, 128, 0.1)",
                        tension: 0.3,
                        fill: true,
                    },
                    {
                        label: "Failed",
                        data: failedData,
                        borderColor: "rgb(248, 113, 113)",
                        backgroundColor: "rgba(248, 113, 113, 0.1)",
                        tension: 0.3,
                        fill: true,
                    },
                ],
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                interaction: {
                    mode: "index",
                    intersect: false,
                },
                onClick: (event, elements) => {
                    if (elements.length > 0) {
                        const index = elements[0].index;
                        const date = analytics.dailyStats[index].date;
                        handleDateDrillDown(date);
                    }
                },
                plugins: {
                    legend: {
                        display: true,
                        position: "top",
                        labels: {
                            color: "rgb(203, 213, 225)",
                            font: {
                                size: 12,
                            },
                        },
                    },
                    tooltip: {
                        backgroundColor: "rgba(15, 23, 42, 0.9)",
                        titleColor: "rgb(226, 232, 240)",
                        bodyColor: "rgb(203, 213, 225)",
                        borderColor: "rgb(71, 85, 105)",
                        borderWidth: 1,
                        callbacks: {
                            afterBody: function () {
                                return "\n\nClick to drill down";
                            },
                        },
                    },
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        ticks: {
                            color: "rgb(148, 163, 184)",
                            stepSize: 1,
                        },
                        grid: {
                            color: "rgba(71, 85, 105, 0.3)",
                        },
                    },
                    x: {
                        ticks: {
                            color: "rgb(148, 163, 184)",
                        },
                        grid: {
                            color: "rgba(71, 85, 105, 0.3)",
                        },
                    },
                },
            },
        });
    }

    function updateWorkspaceChart() {
        // Destroy existing chart if it exists
        if (workspaceChartInstance) {
            workspaceChartInstance.destroy();
        }

        if (
            !workspaceChartCanvas ||
            !analytics?.workspaceStats ||
            analytics.workspaceStats.length === 0
        ) {
            return;
        }

        const ctx = workspaceChartCanvas.getContext("2d");

        // Sort by total jobs descending
        const sortedStats = [...analytics.workspaceStats].sort(
            (a, b) => b.totalJobs - a.totalJobs,
        );

        const labels = sortedStats.map(
            (stat) => stat.workspaceName || stat.workspaceId,
        );
        const successCounts = sortedStats.map((stat) => stat.successful);
        const failureCounts = sortedStats.map((stat) => stat.failed);
        const avgDurations = sortedStats.map((stat) => stat.avgDurationMs);

        workspaceChartInstance = new Chart(ctx, {
            type: "bar",
            data: {
                labels: labels,
                datasets: [
                    {
                        label: "Successful",
                        data: successCounts,
                        backgroundColor: "rgba(74, 222, 128, 0.8)",
                        borderColor: "rgb(74, 222, 128)",
                        borderWidth: 1,
                    },
                    {
                        label: "Failed",
                        data: failureCounts,
                        backgroundColor: "rgba(248, 113, 113, 0.8)",
                        borderColor: "rgb(248, 113, 113)",
                        borderWidth: 1,
                    },
                ],
            },
            options: {
                indexAxis: "y", // This makes it horizontal
                responsive: true,
                maintainAspectRatio: false,
                interaction: {
                    mode: "index",
                    intersect: false,
                },
                onClick: (event, elements) => {
                    if (elements.length > 0) {
                        const index = elements[0].index;
                        const workspaceId = sortedStats[index].workspaceId;
                        const workspaceName = sortedStats[index].workspaceName;
                        handleWorkspaceDrillDown(workspaceId, workspaceName);
                    }
                },
                plugins: {
                    legend: {
                        display: true,
                        position: "top",
                        labels: {
                            color: "rgb(203, 213, 225)",
                            font: {
                                size: 12,
                            },
                        },
                    },
                    tooltip: {
                        backgroundColor: "rgba(15, 23, 42, 0.9)",
                        titleColor: "rgb(226, 232, 240)",
                        bodyColor: "rgb(203, 213, 225)",
                        borderColor: "rgb(71, 85, 105)",
                        borderWidth: 1,
                        callbacks: {
                            label: function (context) {
                                const label = context.dataset.label || "";
                                const value = context.parsed.x || 0;
                                return `${label}: ${value}`;
                            },
                            afterBody: function (context) {
                                const index = context[0].dataIndex;
                                const avgDuration = avgDurations[index];
                                return `\nAvg Duration: ${formatDuration(avgDuration)}\n\nClick to drill down`;
                            },
                        },
                    },
                },
                scales: {
                    x: {
                        beginAtZero: true,
                        ticks: {
                            color: "rgb(148, 163, 184)",
                            stepSize: 1,
                        },
                        grid: {
                            color: "rgba(71, 85, 105, 0.3)",
                        },
                    },
                    y: {
                        ticks: {
                            color: "rgb(148, 163, 184)",
                        },
                        grid: {
                            color: "rgba(71, 85, 105, 0.3)",
                        },
                    },
                },
            },
        });
    }

    function updateJobTypeChart() {
        // Destroy existing chart if it exists
        if (jobTypeChartInstance) {
            jobTypeChartInstance.destroy();
        }

        if (
            !jobTypeChartCanvas ||
            !analytics?.itemTypeStats ||
            analytics.itemTypeStats.length === 0
        ) {
            return;
        }

        const ctx = jobTypeChartCanvas.getContext("2d");

        // Sort by total jobs descending
        const sortedStats = [...analytics.itemTypeStats].sort(
            (a, b) => b.totalJobs - a.totalJobs,
        );

        const labels = sortedStats.map((stat) => stat.itemType || "Unknown");
        const successCounts = sortedStats.map((stat) => stat.successful);
        const failureCounts = sortedStats.map((stat) => stat.failed);
        const avgDurations = sortedStats.map((stat) => stat.avgDurationMs);

        jobTypeChartInstance = new Chart(ctx, {
            type: "bar",
            data: {
                labels: labels,
                datasets: [
                    {
                        label: "Successful",
                        data: successCounts,
                        backgroundColor: "rgba(74, 222, 128, 0.8)",
                        borderColor: "rgb(74, 222, 128)",
                        borderWidth: 1,
                    },
                    {
                        label: "Failed",
                        data: failureCounts,
                        backgroundColor: "rgba(248, 113, 113, 0.8)",
                        borderColor: "rgb(248, 113, 113)",
                        borderWidth: 1,
                    },
                ],
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                interaction: {
                    mode: "index",
                    intersect: false,
                },
                onClick: (event, elements) => {
                    if (elements.length > 0) {
                        const index = elements[0].index;
                        const itemType = sortedStats[index].itemType;
                        handleJobTypeDrillDown(itemType);
                    }
                },
                plugins: {
                    legend: {
                        display: true,
                        position: "top",
                        labels: {
                            color: "rgb(203, 213, 225)",
                            font: {
                                size: 12,
                            },
                        },
                    },
                    tooltip: {
                        backgroundColor: "rgba(15, 23, 42, 0.9)",
                        titleColor: "rgb(226, 232, 240)",
                        bodyColor: "rgb(203, 213, 225)",
                        borderColor: "rgb(71, 85, 105)",
                        borderWidth: 1,
                        callbacks: {
                            label: function (context) {
                                const label = context.dataset.label || "";
                                const value = context.parsed.y || 0;
                                return `${label}: ${value}`;
                            },
                            afterBody: function (context) {
                                const index = context[0].dataIndex;
                                const avgDuration = avgDurations[index];
                                return `\nAvg Duration: ${formatDuration(avgDuration)}\n\nClick to drill down`;
                            },
                        },
                    },
                },
                scales: {
                    x: {
                        ticks: {
                            color: "rgb(148, 163, 184)",
                        },
                        grid: {
                            color: "rgba(71, 85, 105, 0.3)",
                        },
                    },
                    y: {
                        beginAtZero: true,
                        ticks: {
                            color: "rgb(148, 163, 184)",
                            stepSize: 1,
                        },
                        grid: {
                            color: "rgba(71, 85, 105, 0.3)",
                        },
                    },
                },
            },
        });
    }

    onMount(async () => {
        await loadWorkspacesAndItemTypes();
        await loadAnalytics();
        isInitialized = true;
    });

    async function loadWorkspacesAndItemTypes() {
        try {
            // Load all workspaces for the filter dropdown
            allWorkspaces =
                (await window.go.main.App.GetWorkspacesFromCache()) || [];

            // Load available item types based on current filters
            await loadAvailableItemTypes();
        } catch (err) {
            console.error("Failed to load filter data:", err);
        }
    }

    async function loadAvailableItemTypes() {
        try {
            const workspaceIDsArray = Array.from(selectedWorkspaceIds);
            availableItemTypes =
                (await window.go.main.App.GetAvailableItemTypes(
                    selectedDays,
                    workspaceIDsArray,
                )) || [];
        } catch (err) {
            console.error("Failed to load item types:", err);
            availableItemTypes = [];
        }
    }

    async function loadAnalytics() {
        try {
            isLoading = true;
            error = null;

            // Convert Sets to arrays for Go
            const workspaceIDsArray = Array.from(selectedWorkspaceIds);
            const itemTypesArray = Array.from(selectedItemTypes);

            // Use filtered analytics method
            analytics = await window.go.main.App.GetAnalyticsFiltered(
                selectedDays,
                workspaceIDsArray,
                itemTypesArray,
                itemNameSearch,
            );

            console.log("Analytics loaded:", analytics);

            // Log individual sections for debugging
            if (analytics.dailyStatsError) {
                console.error("Daily stats error:", analytics.dailyStatsError);
            }
            if (analytics.workspaceStatsError) {
                console.error(
                    "Workspace stats error:",
                    analytics.workspaceStatsError,
                );
            }
            if (analytics.itemTypeStatsError) {
                console.error(
                    "Item type stats error:",
                    analytics.itemTypeStatsError,
                );
            }

            console.log(
                "Daily stats count:",
                analytics.dailyStats?.length || 0,
            );
            console.log(
                "Workspace stats count:",
                analytics.workspaceStats?.length || 0,
            );
            console.log(
                "Item type stats count:",
                analytics.itemTypeStats?.length || 0,
            );
        } catch (err) {
            console.error("Failed to load analytics:", err);
            error = err.message || "Failed to load analytics";
        } finally {
            isLoading = false;
        }
    }

    // Filter control functions
    function toggleWorkspaceFilter(workspaceId) {
        filterStore.toggleWorkspace(workspaceId);
        // Workspace filter change handled by store subscription which calls loadAnalytics
    }

    function toggleItemTypeFilter(itemType) {
        selectedItemTypes = new Set(selectedItemTypes);
        if (selectedItemTypes.has(itemType)) {
            selectedItemTypes.delete(itemType);
        } else {
            selectedItemTypes.add(itemType);
        }
        // Manually trigger analytics reload
        loadAnalytics();
    }

    function clearAllFilters() {
        filterStore.clearWorkspaces();
        selectedItemTypes = new Set();
        itemNameSearch = "";
        // loadAnalytics will be called by store subscription
    }

    function handleItemNameSearchChange() {
        // Manually trigger analytics reload
        loadAnalytics();
    }

    // Computed filtered workspaces for dropdown
    $: filteredWorkspacesForDropdown = allWorkspaces
        .filter((ws) =>
            (ws.displayName || ws.id)
                .toLowerCase()
                .includes(workspaceSearchText.toLowerCase()),
        )
        .sort((a, b) =>
            (a.displayName || a.id)
                .toLowerCase()
                .localeCompare((b.displayName || b.id).toLowerCase()),
        );

    // Computed filtered item types for dropdown
    $: filteredItemTypesForDropdown = availableItemTypes
        .filter((type) =>
            type.toLowerCase().includes(itemTypeSearchText.toLowerCase()),
        )
        .sort((a, b) => a.toLowerCase().localeCompare(b.toLowerCase()));

    // Count active filters
    $: activeFilterCount =
        selectedWorkspaceIds.size +
        selectedItemTypes.size +
        (itemNameSearch ? 1 : 0);

    async function handleWorkspaceDrillDown(workspaceId, workspaceName) {
        try {
            selectedWorkspace = { id: workspaceId, name: workspaceName };
            selectedJobType = null;
            const result = await window.go.main.App.GetItemStatsByWorkspace(
                workspaceId,
                selectedDays,
            );
            if (result.error) {
                console.error("Error loading workspace items:", result.error);
                error = result.error;
                return;
            }
            drillDownData = result.items;
        } catch (err) {
            console.error("Failed to drill down into workspace:", err);
            error = err.message || "Failed to load workspace items";
        }
    }

    async function handleJobTypeDrillDown(itemType) {
        try {
            selectedJobType = itemType;
            selectedWorkspace = null;
            selectedDate = null;
            const result = await window.go.main.App.GetItemStatsByJobType(
                itemType,
                selectedDays,
            );
            if (result.error) {
                console.error("Error loading job type items:", result.error);
                error = result.error;
                return;
            }
            drillDownData = result.items;
        } catch (err) {
            console.error("Failed to drill down into job type:", err);
            error = err.message || "Failed to load job type items";
        }
    }

    async function handleDateDrillDown(date) {
        try {
            selectedDate = date;
            selectedWorkspace = null;
            selectedJobType = null;

            // Convert Sets to arrays for Go
            const workspaceIDsArray = Array.from(selectedWorkspaceIds);
            const itemTypesArray = Array.from(selectedItemTypes);

            const result = await window.go.main.App.GetItemStatsByDate(
                date,
                workspaceIDsArray,
                itemTypesArray,
                itemNameSearch,
            );
            if (result.error) {
                console.error("Error loading date items:", result.error);
                error = result.error;
                return;
            }
            drillDownData = result.items;
        } catch (err) {
            console.error("Failed to drill down into date:", err);
            error = err.message || "Failed to load date items";
        }
    }

    function closeDrillDown() {
        selectedWorkspace = null;
        selectedJobType = null;
        selectedDate = null;
        drillDownData = null;
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

    function formatDate(dateString) {
        if (!dateString) return "N/A";
        return new Date(dateString).toLocaleDateString();
    }

    function formatDateTime(dateString) {
        if (!dateString) return "N/A";
        return new Date(dateString).toLocaleString();
    }

    function formatPercent(value) {
        if (value === null || value === undefined) return "0%";
        return `${value.toFixed(1)}%`;
    }

    // Close dropdowns when clicking outside
    function handleClickOutside(event) {
        const target = event.target;
        if (!target.closest(".workspace-dropdown-container")) {
            showWorkspaceDropdown = false;
        }
        if (!target.closest(".itemtype-dropdown-container")) {
            showItemTypeDropdown = false;
        }
    }

    // Calculate grand totals for drill-down data
    $: drillDownTotals = drillDownData
        ? drillDownData.reduce(
              (acc, item) => {
                  acc.totalJobs += item.totalJobs || 0;
                  acc.successful += item.successful || 0;
                  acc.failed += item.failed || 0;

                  // For date drill-down with min/max duration
                  if (selectedDate && item.minDurationMs !== undefined) {
                      if (
                          acc.minDurationMs === null ||
                          item.minDurationMs < acc.minDurationMs
                      ) {
                          acc.minDurationMs = item.minDurationMs;
                      }
                      if (
                          acc.maxDurationMs === null ||
                          item.maxDurationMs > acc.maxDurationMs
                      ) {
                          acc.maxDurationMs = item.maxDurationMs;
                      }
                  }

                  // Calculate weighted average for avgDurationMs
                  if (item.avgDurationMs && item.totalJobs) {
                      acc.totalDuration += item.avgDurationMs * item.totalJobs;
                  }

                  return acc;
              },
              {
                  totalJobs: 0,
                  successful: 0,
                  failed: 0,
                  minDurationMs: null,
                  maxDurationMs: null,
                  totalDuration: 0,
                  get successRate() {
                      return this.totalJobs > 0
                          ? (this.successful / this.totalJobs) * 100
                          : 0;
                  },
                  get avgDurationMs() {
                      return this.totalJobs > 0
                          ? this.totalDuration / this.totalJobs
                          : 0;
                  },
              },
          )
        : null;

    $: hasData =
        analytics &&
        !error &&
        (analytics.overallStats?.totalJobs > 0 ||
            analytics.dailyStats?.length > 0);
</script>

<svelte:window on:click={handleClickOutside} />

<div class="h-full overflow-auto bg-slate-900 p-6">
    <!-- Header with Filters -->
    <div class="mb-6">
        <div class="mb-4 flex items-center justify-between">
            <h1 class="text-3xl font-bold text-white">Analytics Dashboard</h1>
            <div class="flex items-center gap-3">
                <label class="text-sm text-slate-300">Time Period:</label>
                <select
                    bind:value={selectedDays}
                    on:change={loadAnalytics}
                    class="rounded-md border border-slate-600 bg-slate-700 px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
                >
                    <option value={1}>Last 24 Hours</option>
                    <option value={7}>Last 7 Days</option>
                    <option value={14}>Last 14 Days</option>
                    <option value={30}>Last 30 Days</option>
                    <option value={90}>Last 90 Days</option>
                </select>
                <button
                    on:click={loadAnalytics}
                    disabled={isLoading}
                    class="rounded-md bg-primary-600 px-4 py-2 text-sm text-white transition-colors hover:bg-primary-700 disabled:opacity-50"
                >
                    {isLoading ? "Loading..." : "Refresh"}
                </button>
            </div>
        </div>

        <!-- Filters Section -->
        <div class="flex items-center gap-3">
            <div class="flex flex-1 gap-3">
                <!-- Workspace Filter -->
                <div class="relative workspace-dropdown-container flex-1">
                    <div class="relative">
                        <button
                            on:click={() =>
                                (showWorkspaceDropdown =
                                    !showWorkspaceDropdown)}
                            class="w-full rounded-md border border-slate-600 bg-slate-700 px-3 py-2 text-left text-sm text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
                        >
                            <span class="mr-2 text-xs text-slate-400"
                                >Workspaces:</span
                            >
                            {#if selectedWorkspaceIds.size === 0}
                                All Workspaces
                            {:else if selectedWorkspaceIds.size === 1}
                                {allWorkspaces.find((ws) =>
                                    selectedWorkspaceIds.has(ws.id),
                                )?.displayName ||
                                    allWorkspaces.find((ws) =>
                                        selectedWorkspaceIds.has(ws.id),
                                    )?.id ||
                                    "1 selected"}
                            {:else}
                                {selectedWorkspaceIds.size} selected
                            {/if}
                            <span class="float-right">▼</span>
                        </button>

                        {#if showWorkspaceDropdown}
                            <div
                                class="absolute z-10 mt-1 w-full rounded-md border border-slate-600 bg-slate-700 shadow-lg"
                            >
                                <div class="p-2">
                                    <input
                                        type="text"
                                        bind:value={workspaceSearchText}
                                        placeholder="Search workspaces..."
                                        class="w-full rounded border border-slate-600 bg-slate-800 px-2 py-1 text-sm text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-primary-500"
                                    />
                                </div>
                                <div class="max-h-60 overflow-y-auto p-2">
                                    {#each filteredWorkspacesForDropdown as workspace}
                                        <label
                                            class="flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-slate-600"
                                        >
                                            <input
                                                type="checkbox"
                                                checked={selectedWorkspaceIds.has(
                                                    workspace.id,
                                                )}
                                                on:change={() =>
                                                    toggleWorkspaceFilter(
                                                        workspace.id,
                                                    )}
                                                class="h-4 w-4 rounded border-slate-500 bg-slate-600 text-primary-600"
                                            />
                                            <span class="text-sm text-white">
                                                {workspace.displayName ||
                                                    workspace.id}
                                            </span>
                                        </label>
                                    {/each}
                                    {#if filteredWorkspacesForDropdown.length === 0}
                                        <p class="p-2 text-sm text-slate-400">
                                            No matching workspaces
                                        </p>
                                    {/if}
                                </div>
                            </div>
                        {/if}
                    </div>
                </div>

                <!-- Item Type Filter -->
                <div class="relative itemtype-dropdown-container flex-1">
                    <div class="relative">
                        <button
                            on:click={() =>
                                (showItemTypeDropdown = !showItemTypeDropdown)}
                            class="w-full rounded-md border border-slate-600 bg-slate-700 px-3 py-2 text-left text-sm text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
                        >
                            <span class="mr-2 text-xs text-slate-400"
                                >Item Types:</span
                            >
                            {#if selectedItemTypes.size === 0}
                                All Types
                            {:else if selectedItemTypes.size === 1}
                                {Array.from(selectedItemTypes)[0]}
                            {:else}
                                {selectedItemTypes.size} selected
                            {/if}
                            <span class="float-right">▼</span>
                        </button>

                        {#if showItemTypeDropdown}
                            <div
                                class="absolute z-10 mt-1 w-full rounded-md border border-slate-600 bg-slate-700 shadow-lg"
                            >
                                <div class="p-2">
                                    <input
                                        type="text"
                                        bind:value={itemTypeSearchText}
                                        placeholder="Search types..."
                                        class="w-full rounded border border-slate-600 bg-slate-800 px-2 py-1 text-sm text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-primary-500"
                                    />
                                </div>
                                <div class="max-h-60 overflow-y-auto p-2">
                                    {#each filteredItemTypesForDropdown as itemType}
                                        <label
                                            class="flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-slate-600"
                                        >
                                            <input
                                                type="checkbox"
                                                checked={selectedItemTypes.has(
                                                    itemType,
                                                )}
                                                on:change={() =>
                                                    toggleItemTypeFilter(
                                                        itemType,
                                                    )}
                                                class="h-4 w-4 rounded border-slate-500 bg-slate-600 text-primary-600"
                                            />
                                            <span class="text-sm text-white">
                                                {itemType}
                                            </span>
                                        </label>
                                    {/each}
                                    {#if filteredItemTypesForDropdown.length === 0}
                                        <p class="p-2 text-sm text-slate-400">
                                            No matching types
                                        </p>
                                    {/if}
                                </div>
                            </div>
                        {/if}
                    </div>
                </div>

                <!-- Item Name Search -->
                <div class="flex-1">
                    <input
                        type="text"
                        bind:value={itemNameSearch}
                        on:input={handleItemNameSearchChange}
                        placeholder="Search by item name..."
                        class="w-full rounded-md border border-slate-600 bg-slate-700 px-3 py-2 text-sm text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-primary-500"
                    />
                </div>
                <button
                    on:click={clearAllFilters}
                    class="rounded-md border border-slate-600 bg-slate-700 px-4 py-2 text-sm text-slate-300 hover:bg-slate-600 hover:text-white transition-colors"
                >
                    Clear All
                </button>
            </div>
        </div>
    </div>

    {#if isLoading}
        <div class="flex items-center justify-center py-20">
            <div class="text-center">
                <div
                    class="mx-auto mb-4 h-12 w-12 animate-spin rounded-full border-b-2 border-primary-500"
                ></div>
                <p class="text-slate-400">Loading analytics...</p>
            </div>
        </div>
    {:else if error}
        <div
            class="rounded-lg bg-red-900/20 border border-red-700 p-6 text-center"
        >
            <svg
                class="mx-auto h-12 w-12 text-red-400 mb-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
            >
                <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                />
            </svg>
            <h3 class="text-lg font-medium text-red-400 mb-2">
                Error Loading Analytics
            </h3>
            <p class="text-slate-300">{error}</p>
        </div>
    {:else if !hasData}
        <div class="rounded-lg bg-slate-800 p-12 text-center">
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
                    d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
                />
            </svg>
            <h3 class="text-xl font-medium text-white mb-2">
                No Data Available
            </h3>
            <p class="text-slate-400">
                No job data found for the selected time period. Try fetching
                data from the API first.
            </p>
        </div>
    {:else}
        <!-- Overall Stats Cards -->
        <div class="mb-6 grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-5">
            <div class="rounded-lg bg-slate-800 p-4 border border-slate-700">
                <div class="text-sm text-slate-400">Total Jobs</div>
                <div class="mt-2 text-3xl font-bold text-white">
                    {analytics.overallStats?.totalJobs || 0}
                </div>
            </div>
            <div class="rounded-lg bg-slate-800 p-4 border border-green-700/30">
                <div class="text-sm text-slate-400">Successful</div>
                <div class="mt-2 text-3xl font-bold text-green-400">
                    {analytics.overallStats?.successful || 0}
                </div>
            </div>
            <div class="rounded-lg bg-slate-800 p-4 border border-red-700/30">
                <div class="text-sm text-slate-400">Failed</div>
                <div class="mt-2 text-3xl font-bold text-red-400">
                    {analytics.overallStats?.failed || 0}
                </div>
            </div>
            <div class="rounded-lg bg-slate-800 p-4 border border-blue-700/30">
                <div class="text-sm text-slate-400">Success Rate</div>
                <div class="mt-2 text-3xl font-bold text-blue-400">
                    {formatPercent(analytics.overallStats?.successRate)}
                </div>
            </div>
            <div class="rounded-lg bg-slate-800 p-4 border border-slate-700">
                <div class="text-sm text-slate-400">Avg Duration</div>
                <div class="mt-2 text-3xl font-bold text-white">
                    {formatDuration(analytics.overallStats?.avgDurationMs)}
                </div>
            </div>
        </div>

        <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
            <!-- Daily Trend -->
            <div class="rounded-lg bg-slate-800 p-6 border border-slate-700">
                <h2 class="mb-4 text-xl font-semibold text-white">
                    Daily Trend
                </h2>
                {#if analytics.dailyStats && analytics.dailyStats.length > 0}
                    <div class="h-64">
                        <canvas bind:this={chartCanvas}></canvas>
                    </div>
                {:else}
                    <p class="text-slate-400">No daily data available</p>
                {/if}
            </div>

            <!-- Workspace Performance -->
            <div class="rounded-lg bg-slate-800 p-6 border border-slate-700">
                <h2 class="mb-4 text-xl font-semibold text-white">
                    Workspace Performance
                </h2>
                {#if analytics.workspaceStats && analytics.workspaceStats.length > 0}
                    <div class="h-64">
                        <canvas bind:this={workspaceChartCanvas}></canvas>
                    </div>
                {:else}
                    <p class="text-slate-400">No workspace data available</p>
                {/if}
            </div>

            <!-- Item Type Breakdown -->
            <div class="rounded-lg bg-slate-800 p-6 border border-slate-700">
                <h2 class="mb-4 text-xl font-semibold text-white">
                    Job Type Breakdown
                </h2>
                {#if analytics.itemTypeStats && analytics.itemTypeStats.length > 0}
                    <div class="h-64">
                        <canvas bind:this={jobTypeChartCanvas}></canvas>
                    </div>
                {:else}
                    <p class="text-slate-400">No item type data available</p>
                {/if}
            </div>

            <!-- Recent Failures -->
            <div class="rounded-lg bg-slate-800 p-6 border border-red-700/30">
                <h2 class="mb-4 text-xl font-semibold text-red-400">
                    Recent Failures
                </h2>
                {#if analytics.recentFailures && analytics.recentFailures.length > 0}
                    <div class="space-y-2 max-h-96 overflow-y-auto">
                        {#each analytics.recentFailures as failure}
                            <div
                                class="rounded-md bg-red-900/20 p-3 border border-red-800/30"
                            >
                                <div class="flex items-center justify-between">
                                    <div
                                        class="text-sm font-medium text-white truncate flex-1"
                                        title={failure.itemDisplayName}
                                    >
                                        {failure.itemDisplayName ||
                                            failure.itemId}
                                    </div>
                                    <FabricLink url={failure.fabricUrl} />
                                </div>
                                <div
                                    class="mt-1 text-xs text-slate-400 truncate"
                                    title={failure.workspaceName}
                                >
                                    {failure.workspaceName ||
                                        failure.workspaceId}
                                </div>
                                <div
                                    class="mt-1 text-xs text-red-300 truncate"
                                    title={failure.failureReason}
                                >
                                    {failure.failureReason ||
                                        "No reason provided"}
                                </div>
                                <div class="mt-1 text-xs text-slate-500">
                                    {formatDateTime(failure.startTime)}
                                </div>
                            </div>
                        {/each}
                    </div>
                {:else}
                    <div class="text-center py-8">
                        <svg
                            class="mx-auto h-12 w-12 text-green-400 mb-2"
                            fill="none"
                            viewBox="0 0 24 24"
                            stroke="currentColor"
                        >
                            <path
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                stroke-width="2"
                                d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                            />
                        </svg>
                        <p class="text-slate-400">No recent failures! 🎉</p>
                    </div>
                {/if}
            </div>
        </div>

        <!-- Drill-down Modal -->
        {#if drillDownData}
            <div
                class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
            >
                <div
                    class="max-h-[90vh] w-full max-w-7xl overflow-auto rounded-lg bg-slate-800 border border-slate-700 shadow-2xl"
                >
                    <!-- Modal Header -->
                    <div
                        class="sticky top-0 z-10 flex items-center justify-between border-b border-slate-700 bg-slate-800 p-4"
                    >
                        <h2 class="text-xl font-semibold text-white">
                            {#if selectedWorkspace}
                                Items in {selectedWorkspace.name}
                            {:else if selectedJobType}
                                {selectedJobType} Items
                            {:else if selectedDate}
                                Items on {formatDate(selectedDate)}
                            {/if}
                        </h2>
                        <button
                            on:click={closeDrillDown}
                            class="rounded-md p-2 text-slate-400 transition-colors hover:bg-slate-700 hover:text-white"
                        >
                            <svg
                                class="h-6 w-6"
                                fill="none"
                                viewBox="0 0 24 24"
                                stroke="currentColor"
                            >
                                <path
                                    stroke-linecap="round"
                                    stroke-linejoin="round"
                                    stroke-width="2"
                                    d="M6 18L18 6M6 6l12 12"
                                />
                            </svg>
                        </button>
                    </div>

                    <!-- Modal Content -->
                    <div class="p-4">
                        {#if drillDownData.length === 0}
                            <p class="py-8 text-center text-slate-400">
                                No items found
                            </p>
                        {:else}
                            <div class="overflow-x-auto">
                                <table class="w-full">
                                    <thead class="bg-slate-700">
                                        <tr>
                                            <th
                                                class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-300"
                                                >Workspace</th
                                            >
                                            <th
                                                class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-300"
                                                >Item</th
                                            >
                                            <th
                                                class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-300"
                                                >Type</th
                                            >
                                            <th
                                                class="px-4 py-3 text-center text-xs font-medium uppercase tracking-wider text-slate-300"
                                                >Total</th
                                            >
                                            <th
                                                class="px-4 py-3 text-center text-xs font-medium uppercase tracking-wider text-slate-300"
                                                >Success</th
                                            >
                                            <th
                                                class="px-4 py-3 text-center text-xs font-medium uppercase tracking-wider text-slate-300"
                                                >Failed</th
                                            >
                                            <th
                                                class="px-4 py-3 text-center text-xs font-medium uppercase tracking-wider text-slate-300"
                                                >Success Rate</th
                                            >
                                            {#if selectedDate}
                                                <th
                                                    class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-slate-300"
                                                    >Min Duration</th
                                                >
                                                <th
                                                    class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-slate-300"
                                                    >Max Duration</th
                                                >
                                            {/if}
                                            <th
                                                class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-slate-300"
                                                >Avg Duration</th
                                            >
                                        </tr>
                                    </thead>
                                    <tbody class="divide-y divide-slate-700">
                                        {#each drillDownData as item}
                                            <tr class="hover:bg-slate-700/50">
                                                <td class="px-4 py-3">
                                                    <div
                                                        class="text-sm text-slate-300 truncate"
                                                        title={item.workspaceName}
                                                    >
                                                        {item.workspaceName ||
                                                            item.workspaceId ||
                                                            "N/A"}
                                                    </div>
                                                </td>
                                                <td class="px-4 py-3">
                                                    <div
                                                        class="text-sm text-white truncate"
                                                        title={item.itemName}
                                                    >
                                                        {item.itemName ||
                                                            item.itemId}
                                                    </div>
                                                </td>
                                                <td
                                                    class="px-4 py-3 text-sm text-slate-300"
                                                >
                                                    {item.itemType || "N/A"}
                                                </td>
                                                <td
                                                    class="px-4 py-3 text-center text-sm font-medium text-white"
                                                >
                                                    {item.totalJobs}
                                                </td>
                                                <td
                                                    class="px-4 py-3 text-center text-sm font-medium text-green-400"
                                                >
                                                    {item.successful}
                                                </td>
                                                <td
                                                    class="px-4 py-3 text-center text-sm font-medium text-red-400"
                                                >
                                                    {item.failed}
                                                </td>
                                                <td
                                                    class="px-4 py-3 text-center text-sm font-medium text-blue-400"
                                                >
                                                    {formatPercent(
                                                        item.successRate,
                                                    )}
                                                </td>
                                                {#if selectedDate}
                                                    <td
                                                        class="px-4 py-3 text-right text-sm text-slate-300"
                                                    >
                                                        {formatDuration(
                                                            item.minDurationMs,
                                                        )}
                                                    </td>
                                                    <td
                                                        class="px-4 py-3 text-right text-sm text-slate-300"
                                                    >
                                                        {formatDuration(
                                                            item.maxDurationMs,
                                                        )}
                                                    </td>
                                                {/if}
                                                <td
                                                    class="px-4 py-3 text-right text-sm text-slate-300"
                                                >
                                                    {formatDuration(
                                                        item.avgDurationMs,
                                                    )}
                                                </td>
                                            </tr>
                                        {/each}
                                    </tbody>
                                    <tfoot
                                        class="bg-slate-700/50 border-t-2 border-slate-600"
                                    >
                                        <tr class="font-semibold">
                                            <td
                                                class="px-4 py-3 text-sm text-white"
                                                colspan="3"
                                            >
                                                Grand Total
                                            </td>
                                            <td
                                                class="px-4 py-3 text-center text-sm font-bold text-white"
                                            >
                                                {drillDownTotals?.totalJobs ||
                                                    0}
                                            </td>
                                            <td
                                                class="px-4 py-3 text-center text-sm font-bold text-green-400"
                                            >
                                                {drillDownTotals?.successful ||
                                                    0}
                                            </td>
                                            <td
                                                class="px-4 py-3 text-center text-sm font-bold text-red-400"
                                            >
                                                {drillDownTotals?.failed || 0}
                                            </td>
                                            <td
                                                class="px-4 py-3 text-center text-sm font-bold text-blue-400"
                                            >
                                                {formatPercent(
                                                    drillDownTotals?.successRate ||
                                                        0,
                                                )}
                                            </td>
                                            {#if selectedDate}
                                                <td
                                                    class="px-4 py-3 text-right text-sm font-bold text-slate-300"
                                                >
                                                    {formatDuration(
                                                        drillDownTotals?.minDurationMs,
                                                    )}
                                                </td>
                                                <td
                                                    class="px-4 py-3 text-right text-sm font-bold text-slate-300"
                                                >
                                                    {formatDuration(
                                                        drillDownTotals?.maxDurationMs,
                                                    )}
                                                </td>
                                            {/if}
                                            <td
                                                class="px-4 py-3 text-right text-sm font-bold text-slate-300"
                                            >
                                                {formatDuration(
                                                    drillDownTotals?.avgDurationMs ||
                                                        0,
                                                )}
                                            </td>
                                        </tr>
                                    </tfoot>
                                </table>
                            </div>
                        {/if}
                    </div>
                </div>
            </div>
        {/if}

        <!-- Long Running Jobs -->
        {#if analytics.longRunningJobs && analytics.longRunningJobs.length > 0}
            <div
                class="mt-6 rounded-lg bg-slate-800 p-6 border border-yellow-700/30"
            >
                <h2 class="mb-4 text-xl font-semibold text-yellow-400">
                    Long Running Jobs (50%+ above average)
                </h2>
                <div class="overflow-x-auto">
                    <table class="w-full">
                        <thead class="bg-slate-700">
                            <tr>
                                <th
                                    class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-300"
                                    >Job</th
                                >
                                <th
                                    class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-300"
                                    >Workspace</th
                                >
                                <th
                                    class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-300"
                                    >Duration</th
                                >
                                <th
                                    class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-300"
                                    >Average</th
                                >
                                <th
                                    class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-300"
                                    >Deviation</th
                                >
                                <th
                                    class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-300"
                                    >Started</th
                                >
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-slate-700">
                            {#each analytics.longRunningJobs as job}
                                <tr class="hover:bg-slate-700/50">
                                    <td class="px-4 py-3">
                                        <div class="flex items-center gap-2">
                                            <div class="flex-1">
                                                <div
                                                    class="text-sm text-white truncate"
                                                    title={job.itemDisplayName}
                                                >
                                                    {job.itemDisplayName ||
                                                        job.itemId}
                                                </div>
                                                <div
                                                    class="text-xs text-slate-400"
                                                >
                                                    {job.itemType || "N/A"}
                                                </div>
                                            </div>
                                            <FabricLink url={job.fabricUrl} />
                                        </div>
                                    </td>
                                    <td
                                        class="px-4 py-3 text-sm text-slate-300 truncate"
                                        title={job.workspaceName}
                                    >
                                        {job.workspaceName || job.workspaceId}
                                    </td>
                                    <td
                                        class="px-4 py-3 text-sm font-medium text-yellow-400"
                                    >
                                        {formatDuration(job.durationMs)}
                                    </td>
                                    <td
                                        class="px-4 py-3 text-sm text-slate-400"
                                    >
                                        {formatDuration(job.avgDurationMs)}
                                    </td>
                                    <td
                                        class="px-4 py-3 text-sm font-bold text-yellow-400"
                                    >
                                        +{job.deviationPct.toFixed(0)}%
                                    </td>
                                    <td
                                        class="px-4 py-3 text-sm text-slate-400"
                                    >
                                        {formatDateTime(job.startTime)}
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            </div>
        {/if}
    {/if}
</div>
