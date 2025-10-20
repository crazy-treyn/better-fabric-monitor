<script>
    import { authStore, authActions } from "../stores/auth.js";

    let tenantId = "";
    let isLoading = false;

    async function handleLogin() {
        if (!tenantId.trim()) {
            authStore.update((state) => ({
                ...state,
                error: "Please enter your Azure Tenant ID",
            }));
            return;
        }

        isLoading = true;
        try {
            await authActions.login(tenantId.trim());
        } finally {
            isLoading = false;
        }
    }

    function handleKeyPress(event) {
        if (event.key === "Enter") {
            handleLogin();
        }
    }
</script>

<div class="min-h-screen flex items-center justify-center bg-slate-900 px-4">
    <div class="max-w-md w-full space-y-8">
        <!-- Header -->
        <div class="text-center">
            <div
                class="mx-auto h-16 w-16 bg-primary-500 rounded-full flex items-center justify-center mb-4"
            >
                <svg
                    class="h-8 w-8 text-white"
                    fill="currentColor"
                    viewBox="0 0 24 24"
                >
                    <path
                        d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"
                    />
                </svg>
            </div>
            <h2 class="text-3xl font-bold text-white mb-2">Fabric Monitor</h2>
            <p class="text-slate-400">
                Monitor your Microsoft Fabric workspaces
            </p>
        </div>

        <!-- Login Form -->
        <div class="bg-slate-800 rounded-lg shadow-lg p-8">
            <form on:submit|preventDefault={handleLogin} class="space-y-6">
                <div>
                    <label
                        for="tenantId"
                        class="block text-sm font-medium text-slate-300 mb-2"
                    >
                        Azure Tenant ID
                    </label>
                    <input
                        id="tenantId"
                        type="text"
                        bind:value={tenantId}
                        on:keypress={handleKeyPress}
                        placeholder="Enter your Azure tenant ID (e.g., contoso.onmicrosoft.com or GUID)"
                        class="w-full px-3 py-2 border border-slate-600 rounded-md shadow-sm bg-slate-700 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                        disabled={isLoading}
                        required
                    />
                    <p class="mt-1 text-sm text-slate-400">
                        You can find this in your Azure portal under Azure
                        Active Directory → Properties
                    </p>
                </div>

                {#if $authStore.error}
                    <div
                        class="bg-red-900/50 border border-red-500 rounded-md p-3"
                    >
                        <p class="text-sm text-red-400">{$authStore.error}</p>
                    </div>
                {/if}

                {#if $authStore.deviceCode}
                    <div
                        class="bg-primary-900/50 border border-primary-500 rounded-md p-4 space-y-3"
                    >
                        <div class="text-center">
                            <p class="text-sm text-slate-300 mb-2">
                                Enter this code in the browser window that just
                                opened:
                            </p>
                            <div
                                class="text-3xl font-mono font-bold text-primary-400 bg-slate-900 rounded-lg py-3 px-4 mb-2"
                            >
                                {$authStore.deviceCode.userCode}
                            </div>
                            <button
                                type="button"
                                on:click={() =>
                                    navigator.clipboard.writeText(
                                        $authStore.deviceCode.userCode,
                                    )}
                                class="text-sm text-primary-400 hover:text-primary-300 underline"
                            >
                                Copy code
                            </button>
                        </div>
                        <div class="text-xs text-slate-400 text-center">
                            <p>Or visit:</p>
                            <a
                                href={$authStore.deviceCode.verificationURL}
                                target="_blank"
                                class="text-primary-400 hover:text-primary-300 underline break-all"
                            >
                                {$authStore.deviceCode.verificationURL}
                            </a>
                        </div>
                        {#if $authStore.isWaitingForCode}
                            <div class="flex items-center justify-center gap-2">
                                <svg
                                    class="animate-spin h-4 w-4 text-primary-400"
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
                                <span class="text-sm text-slate-400"
                                    >Waiting for authentication...</span
                                >
                            </div>
                        {/if}
                    </div>
                {/if}

                <button
                    type="submit"
                    disabled={isLoading || $authStore.isWaitingForCode}
                    class="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-primary-600 hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                    {#if isLoading || $authStore.isWaitingForCode}
                        <svg
                            class="animate-spin -ml-1 mr-3 h-5 w-5 text-white"
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
                        Signing in...
                    {:else}
                        Sign in with Microsoft
                    {/if}
                </button>
            </form>

            <!-- Help Text -->
            <div class="mt-6 text-center">
                <p class="text-sm text-slate-400">
                    This app requires Fabric.ReadWrite.All permissions in your
                    Azure app registration.
                </p>
            </div>
        </div>
    </div>
</div>
