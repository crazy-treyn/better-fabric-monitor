<script>
    import { onMount } from "svelte";
    import { authStore, authActions } from "../stores/auth.js";

    let workspaces = [];
    let jobs = [];
    let isLoading = true;

    onMount(async () => {
        await loadData();
    });

    async function loadData() {
        try {
            isLoading = true;
            // Load workspaces and jobs from backend
            workspaces = (await window.go.main.App.GetWorkspaces()) || [];
            jobs = (await window.go.main.App.GetJobs()) || [];
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
</script>

<div class="h-screen flex flex-col bg-slate-900">
    <!-- Header -->
    <header class="bg-slate-800 border-b border-slate-700 px-6 py-4">
        <div class="flex items-center justify-between">
            <div class="flex items-center space-x-4">
                <h1 class="text-xl font-semibold text-white">Fabric Monitor</h1>
                <span class="text-sm text-slate-400"
                    >Tenant: {$authStore.tenantId}</span
                >
            </div>
            <button
                on:click={handleLogout}
                class="px-4 py-2 text-sm text-slate-300 hover:text-white hover:bg-slate-700 rounded-md transition-colors"
            >
                Sign Out
            </button>
        </div>
    </header>

    <!-- Main Content -->
    <main class="flex-1 overflow-hidden">
        {#if isLoading}
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
                    class="w-64 bg-slate-800 border-r border-slate-700 p-4 flex flex-col"
                >
                    <h2 class="text-lg font-semibold text-white mb-4">
                        Workspaces
                    </h2>
                    <div class="space-y-2 overflow-y-auto flex-1">
                        {#each workspaces as workspace}
                            <div
                                class="p-3 bg-slate-700 rounded-lg cursor-pointer hover:bg-slate-600 transition-colors"
                            >
                                <h3 class="text-sm font-medium text-white">
                                    {workspace.displayName || workspace.id}
                                </h3>
                                <p class="text-xs text-slate-400">
                                    {workspace.type}
                                </p>
                            </div>
                        {/each}
                        {#if workspaces.length === 0}
                            <p class="text-slate-400 text-sm">
                                No workspaces found
                            </p>
                        {/if}
                    </div>
                </div>

                <!-- Main Panel -->
                <div class="flex-1 p-6 overflow-auto">
                    <div class="mb-6">
                        <h2 class="text-2xl font-bold text-white mb-4">
                            Recent Jobs
                        </h2>

                        {#if jobs.length > 0}
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
                                        {#each jobs.slice(0, 50) as job}
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
                                                    {job.durationMs
                                                        ? `${Math.round(job.durationMs / 1000)}s`
                                                        : "N/A"}
                                                </td>
                                            </tr>
                                        {/each}
                                    </tbody>
                                </table>
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
                                    No jobs found
                                </h3>
                                <p class="text-slate-400">
                                    Jobs will appear here once they start
                                    running in your workspaces.
                                </p>
                            </div>
                        {/if}
                    </div>
                </div>
            </div>
        {/if}
    </main>
</div>
