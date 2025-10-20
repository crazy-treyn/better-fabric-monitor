<script>
  import { onMount } from "svelte";
  import { authStore, authActions } from "./stores/auth.js";
  import LoginView from "./components/LoginView.svelte";
  import Dashboard from "./components/Dashboard.svelte";

  let currentView = "login";
  let isCheckingAuth = true;

  onMount(async () => {
    // Check if user is already authenticated from cache
    await authActions.checkAuth();
    isCheckingAuth = false;

    if ($authStore.isAuthenticated) {
      currentView = "dashboard";
    }
  });

  // Listen for auth state changes
  $: if ($authStore.isAuthenticated) {
    currentView = "dashboard";
  } else if (!isCheckingAuth) {
    currentView = "login";
  }
</script>

<main class="h-screen flex flex-col">
  {#if currentView === "login"}
    <LoginView />
  {:else if currentView === "dashboard"}
    <Dashboard />
  {/if}
</main>
