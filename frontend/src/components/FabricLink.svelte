<script>
    import { BrowserOpenURL } from "../../wailsjs/runtime/runtime";

    export let url = "";
    export let title = "View in Microsoft Fabric";

    // Only show if URL is provided and not empty
    $: showLink = url && url.trim() !== "";

    function openLink() {
        if (url) {
            BrowserOpenURL(url);
        }
    }
</script>

{#if showLink}
    <button
        on:click={openLink}
        {title}
        class="inline-flex items-center text-primary-400 hover:text-primary-300 transition-colors ml-2 cursor-pointer bg-transparent border-0 p-0"
    >
        <!-- External link icon -->
        <svg
            class="h-4 w-4"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
        >
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
            />
        </svg>
    </button>
{/if}
