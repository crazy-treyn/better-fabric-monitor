import { writable } from "svelte/store";

/**
 * Shared filter store for syncing workspace selection between Jobs and Analytics tabs
 */
function createFilterStore() {
    const { subscribe, set, update } = writable({
        selectedWorkspaceIds: new Set(), // Set of workspace IDs
    });

    return {
        subscribe,

        // Toggle a single workspace ID
        toggleWorkspace: (workspaceId) => {
            update((state) => {
                const newIds = new Set(state.selectedWorkspaceIds);
                if (newIds.has(workspaceId)) {
                    newIds.delete(workspaceId);
                } else {
                    newIds.add(workspaceId);
                }
                return { ...state, selectedWorkspaceIds: newIds };
            });
        },

        // Set multiple workspace IDs
        setWorkspaces: (workspaceIds) => {
            update((state) => ({
                ...state,
                selectedWorkspaceIds: new Set(workspaceIds),
            }));
        },

        // Clear all workspace selections
        clearWorkspaces: () => {
            update((state) => ({
                ...state,
                selectedWorkspaceIds: new Set(),
            }));
        },

        // Select all workspaces from a given list
        selectAllWorkspaces: (workspaces) => {
            const allIds = workspaces.map((ws) => ws.id);
            update((state) => ({
                ...state,
                selectedWorkspaceIds: new Set(allIds),
            }));
        },

        // Reset entire store
        reset: () => {
            set({
                selectedWorkspaceIds: new Set(),
            });
        },
    };
}

export const filterStore = createFilterStore();
