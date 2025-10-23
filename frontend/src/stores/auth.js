import { writable } from 'svelte/store';

// Auth store for managing authentication state
export const authStore = writable({
    isAuthenticated: false,
    user: null,
    tenantId: '',
    error: null,
    deviceCode: null,
    isWaitingForCode: false,
    offlineMode: false
});

// Auth actions
export const authActions = {
    async login(tenantId) {
        try {
            authStore.update(state => ({ ...state, error: null, isWaitingForCode: false }));

            // Call backend to start device code flow
            const result = await window.go.main.App.Login(tenantId);

            if (result.success && result.requiresCode) {
                // Device code flow - show the code to user
                authStore.update(state => ({
                    ...state,
                    deviceCode: {
                        userCode: result.userCode,
                        verificationURL: result.verificationURL,
                        message: result.message
                    },
                    tenantId: tenantId,
                    isWaitingForCode: true,
                    error: null
                }));

                // Start polling for completion
                this.completeLogin();
            } else if (result.success) {
                // Direct success (shouldn't happen with device code flow)
                authStore.update(state => ({
                    ...state,
                    isAuthenticated: true,
                    user: result.user,
                    tenantId: tenantId,
                    deviceCode: null,
                    isWaitingForCode: false,
                    error: null
                }));
            } else {
                authStore.update(state => ({
                    ...state,
                    error: result.error || 'Login failed',
                    isWaitingForCode: false
                }));
            }
        } catch (error) {
            authStore.update(state => ({
                ...state,
                error: error.message || 'Login failed',
                isWaitingForCode: false
            }));
        }
    },

    async completeLogin() {
        try {
            // Wait for user to complete authentication
            const result = await window.go.main.App.CompleteLogin();

            if (result.success) {
                authStore.update(state => ({
                    ...state,
                    isAuthenticated: true,
                    user: result.user,
                    deviceCode: null,
                    isWaitingForCode: false,
                    error: null
                }));
            } else {
                authStore.update(state => ({
                    ...state,
                    error: result.error || 'Authentication failed',
                    deviceCode: null,
                    isWaitingForCode: false
                }));
            }
        } catch (error) {
            authStore.update(state => ({
                ...state,
                error: error.message || 'Authentication failed',
                deviceCode: null,
                isWaitingForCode: false
            }));
        }
    },

    async logout() {
        try {
            await window.go.main.App.Logout();
            authStore.set({
                isAuthenticated: false,
                user: null,
                tenantId: '',
                error: null,
                deviceCode: null,
                isWaitingForCode: false,
                offlineMode: false
            });
        } catch (error) {
            console.error('Logout error:', error);
        }
    },

    continueOffline() {
        authStore.update(state => ({
            ...state,
            offlineMode: true,
            isAuthenticated: true, // Mark as "authenticated" for UI purposes
            error: null,
            deviceCode: null,
            isWaitingForCode: false
        }));
    },

    exitOfflineMode() {
        authStore.update(state => ({
            ...state,
            offlineMode: false,
            isAuthenticated: false
        }));
    },

    async checkAuth() {
        try {
            const isAuthenticated = await window.go.main.App.IsAuthenticated();
            if (isAuthenticated) {
                // Get user info if authenticated
                const user = await window.go.main.App.GetUserInfo();
                authStore.update(state => ({
                    ...state,
                    isAuthenticated: true,
                    user: user
                }));
            }
        } catch (error) {
            console.error('Auth check error:', error);
        }
    }
};