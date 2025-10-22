<script>
    import { onMount, onDestroy } from "svelte";

    let logs = [];
    let isLoading = true;
    let autoRefresh = true;
    let autoScroll = true;
    let refreshInterval;
    let filterLevel = "all"; // all, INFO, WARNING, ERROR, DEBUG
    let searchText = "";
    let logsContainer;
    let appVersion = "";

    onMount(async () => {
        await loadLogs();
        await loadVersion();
        if (autoRefresh) {
            startAutoRefresh();
        }
    });

    onDestroy(() => {
        stopAutoRefresh();
    });

    async function loadVersion() {
        try {
            appVersion = await window.go.main.App.GetAppVersion();
        } catch (error) {
            console.error("Failed to load version:", error);
            appVersion = "unknown";
        }
    }

    async function loadLogs() {
        try {
            isLoading = true;
            logs = (await window.go.main.App.GetLogs()) || [];
        } catch (error) {
            console.error("Failed to load logs:", error);
        } finally {
            isLoading = false;
            if (autoScroll && logsContainer) {
                setTimeout(() => {
                    logsContainer.scrollTop = logsContainer.scrollHeight;
                }, 100);
            }
        }
    }

    async function clearLogs() {
        try {
            await window.go.main.App.ClearLogs();
            await loadLogs();
        } catch (error) {
            console.error("Failed to clear logs:", error);
        }
    }

    function startAutoRefresh() {
        if (refreshInterval) {
            clearInterval(refreshInterval);
        }
        refreshInterval = setInterval(loadLogs, 2000); // Refresh every 2 seconds
    }

    function stopAutoRefresh() {
        if (refreshInterval) {
            clearInterval(refreshInterval);
            refreshInterval = null;
        }
    }

    function toggleAutoRefresh() {
        autoRefresh = !autoRefresh;
        if (autoRefresh) {
            startAutoRefresh();
            loadLogs(); // Immediate refresh
        } else {
            stopAutoRefresh();
        }
    }

    function copyToClipboard() {
        const logText = filteredLogs
            .map(
                (log) =>
                    `[${formatTimestamp(log.timestamp)}] ${log.level}: ${log.message}`,
            )
            .join("\n");
        navigator.clipboard.writeText(logText);
    }

    function downloadLogs() {
        const logText = filteredLogs
            .map(
                (log) =>
                    `[${formatTimestamp(log.timestamp)}] ${log.level}: ${log.message}`,
            )
            .join("\n");
        const blob = new Blob([logText], { type: "text/plain" });
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `fabric-monitor-logs-${new Date().toISOString().split("T")[0]}.txt`;
        a.click();
        URL.revokeObjectURL(url);
    }

    function formatTimestamp(timestamp) {
        return new Date(timestamp).toLocaleTimeString("en-US", {
            hour12: false,
            hour: "2-digit",
            minute: "2-digit",
            second: "2-digit",
            fractionalSecondDigits: 3,
        });
    }

    function getLevelColor(level) {
        switch (level) {
            case "ERROR":
                return "text-red-400 bg-red-900/30";
            case "WARNING":
                return "text-yellow-400 bg-yellow-900/30";
            case "DEBUG":
                return "text-slate-400 bg-slate-700/30";
            default:
                return "text-blue-400 bg-blue-900/30";
        }
    }

    function getLevelBadgeColor(level) {
        switch (level) {
            case "ERROR":
                return "bg-red-900/50 text-red-300 border-red-700";
            case "WARNING":
                return "bg-yellow-900/50 text-yellow-300 border-yellow-700";
            case "DEBUG":
                return "bg-slate-700/50 text-slate-300 border-slate-600";
            default:
                return "bg-blue-900/50 text-blue-300 border-blue-700";
        }
    }

    // Computed filtered logs
    $: filteredLogs = logs.filter((log) => {
        const matchesLevel = filterLevel === "all" || log.level === filterLevel;
        const matchesSearch =
            !searchText ||
            log.message.toLowerCase().includes(searchText.toLowerCase());
        return matchesLevel && matchesSearch;
    });

    // Count by level
    $: levelCounts = logs.reduce(
        (acc, log) => {
            acc[log.level] = (acc[log.level] || 0) + 1;
            return acc;
        },
        { INFO: 0, WARNING: 0, ERROR: 0, DEBUG: 0 },
    );
</script>

<div class="h-full flex flex-col bg-slate-900 p-6">
    <!-- Header -->
    <div class="mb-4">
        <div class="flex items-center justify-between mb-4">
            <div>
                <h2 class="text-2xl font-bold text-white">Application Logs</h2>
                <p class="text-sm text-slate-400 mt-1">
                    Real-time log messages from the backend (last 2000 entries)
                </p>
            </div>
            <div class="flex gap-2">
                <button
                    on:click={toggleAutoRefresh}
                    class="px-4 py-2 text-sm rounded-md transition-colors {autoRefresh
                        ? 'bg-primary-600 text-white'
                        : 'bg-slate-700 text-slate-300 hover:bg-slate-600'}"
                    title="{autoRefresh
                        ? 'Disable'
                        : 'Enable'} auto-refresh (every 2s)"
                >
                    {autoRefresh ? "🔄 Auto-refresh ON" : "⏸️ Auto-refresh OFF"}
                </button>
                <button
                    on:click={() => (autoScroll = !autoScroll)}
                    class="px-4 py-2 text-sm rounded-md transition-colors {autoScroll
                        ? 'bg-primary-600 text-white'
                        : 'bg-slate-700 text-slate-300 hover:bg-slate-600'}"
                    title="{autoScroll
                        ? 'Disable'
                        : 'Enable'} auto-scroll to bottom"
                >
                    {autoScroll ? "📍 Auto-scroll ON" : "📍 Auto-scroll OFF"}
                </button>
                <button
                    on:click={copyToClipboard}
                    class="px-4 py-2 text-sm bg-slate-700 hover:bg-slate-600 text-white rounded-md transition-colors"
                    title="Copy all logs to clipboard"
                >
                    📋 Copy
                </button>
                <button
                    on:click={downloadLogs}
                    class="px-4 py-2 text-sm bg-slate-700 hover:bg-slate-600 text-white rounded-md transition-colors"
                    title="Download logs as text file"
                >
                    💾 Download
                </button>
                <button
                    on:click={clearLogs}
                    class="px-4 py-2 text-sm bg-red-600 hover:bg-red-700 text-white rounded-md transition-colors"
                    title="Clear all logs"
                >
                    🧹 Clear
                </button>
            </div>
        </div>

        <!-- Filters and Stats -->
        <div class="flex items-center gap-4">
            <div class="flex-1">
                <input
                    type="text"
                    bind:value={searchText}
                    placeholder="Search logs..."
                    class="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
            </div>
            <div class="flex gap-2">
                <button
                    on:click={() => (filterLevel = "all")}
                    class="px-3 py-2 text-xs rounded-md transition-colors {filterLevel ===
                    'all'
                        ? 'bg-primary-600 text-white'
                        : 'bg-slate-700 text-slate-300 hover:bg-slate-600'}"
                >
                    All ({logs.length})
                </button>
                <button
                    on:click={() => (filterLevel = "INFO")}
                    class="px-3 py-2 text-xs rounded-md transition-colors {filterLevel ===
                    'INFO'
                        ? 'bg-blue-600 text-white'
                        : 'bg-slate-700 text-slate-300 hover:bg-slate-600'}"
                >
                    🔵 INFO ({levelCounts.INFO || 0})
                </button>
                <button
                    on:click={() => (filterLevel = "WARNING")}
                    class="px-3 py-2 text-xs rounded-md transition-colors {filterLevel ===
                    'WARNING'
                        ? 'bg-yellow-600 text-white'
                        : 'bg-slate-700 text-slate-300 hover:bg-slate-600'}"
                >
                    🟡 WARNING ({levelCounts.WARNING || 0})
                </button>
                <button
                    on:click={() => (filterLevel = "ERROR")}
                    class="px-3 py-2 text-xs rounded-md transition-colors {filterLevel ===
                    'ERROR'
                        ? 'bg-red-600 text-white'
                        : 'bg-slate-700 text-slate-300 hover:bg-slate-600'}"
                >
                    🔴 ERROR ({levelCounts.ERROR || 0})
                </button>
                <button
                    on:click={() => (filterLevel = "DEBUG")}
                    class="px-3 py-2 text-xs rounded-md transition-colors {filterLevel ===
                    'DEBUG'
                        ? 'bg-slate-500 text-white'
                        : 'bg-slate-700 text-slate-300 hover:bg-slate-600'}"
                >
                    ⚪ DEBUG ({levelCounts.DEBUG || 0})
                </button>
            </div>
        </div>
    </div>

    <!-- Logs Container -->
    <div
        bind:this={logsContainer}
        class="flex-1 bg-slate-800 rounded-lg border border-slate-700 overflow-y-auto font-mono text-sm"
    >
        {#if isLoading && logs.length === 0}
            <div class="flex items-center justify-center h-full">
                <div class="text-center">
                    <div
                        class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-500 mx-auto mb-4"
                    ></div>
                    <p class="text-slate-400">Loading logs...</p>
                </div>
            </div>
        {:else if filteredLogs.length === 0}
            <div class="flex items-center justify-center h-full">
                <div class="text-center">
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
                            d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                        />
                    </svg>
                    <h3 class="text-lg font-medium text-white mb-2">
                        {logs.length === 0 ? "No logs yet" : "No matching logs"}
                    </h3>
                    <p class="text-slate-400">
                        {logs.length === 0
                            ? "Logs will appear here as operations are performed."
                            : "Try adjusting your filters or search terms."}
                    </p>
                </div>
            </div>
        {:else}
            <div class="p-2 space-y-0.5">
                {#each filteredLogs as log}
                    <div
                        class="px-3 py-1.5 rounded hover:bg-slate-700/50 transition-colors {getLevelColor(
                            log.level,
                        )}"
                    >
                        <div class="flex items-start gap-2">
                            <span
                                class="text-slate-400 text-xs w-24 flex-shrink-0"
                                >{formatTimestamp(log.timestamp)}</span
                            >
                            <span
                                class="px-2 py-0.5 text-xs font-semibold rounded border {getLevelBadgeColor(
                                    log.level,
                                )} w-20 text-center flex-shrink-0"
                            >
                                {log.level}
                            </span>
                            <span class="text-slate-200 break-all flex-1"
                                >{log.message}</span
                            >
                        </div>
                    </div>
                {/each}
            </div>
        {/if}
    </div>

    <!-- Footer Stats -->
    <div class="mt-3 text-sm text-slate-400 flex items-center justify-between">
        <div class="flex items-center gap-4">
            <span>
                Showing {filteredLogs.length} of {logs.length} log entries
            </span>
            <span class="text-xs text-slate-400">
                • v{appVersion || "loading..."}
            </span>
        </div>
        {#if autoRefresh}
            <div class="flex items-center gap-2">
                <div
                    class="w-2 h-2 rounded-full bg-green-500 animate-pulse"
                ></div>
                <span>Live updates enabled</span>
            </div>
        {/if}
    </div>
</div>
