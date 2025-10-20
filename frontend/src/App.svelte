<script>
  import { onMount } from "svelte";
  import { authStore } from "./stores/auth.js";
  import LoginView from "./components/LoginView.svelte";
  import Dashboard from "./components/Dashboard.svelte";

  let currentView = "login";

  onMount(() => {
    // Check if user is already authenticated
    if ($authStore.isAuthenticated) {
      currentView = "dashboard";
    }
  });

  // Listen for auth state changes
  $: if ($authStore.isAuthenticated) {
    currentView = "dashboard";
  } else {
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
