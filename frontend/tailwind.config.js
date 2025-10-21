/** @type {import('tailwindcss').Config} */
export default {
    content: [
        "./src/**/*.{js,ts,jsx,tsx,svelte}",
    ],
    theme: {
        extend: {
            colors: {
                primary: {
                    50: '#eff6ff',
                    100: '#dbeafe',
                    200: '#bfdbfe',
                    300: '#93c5fd',
                    400: '#60a5fa',
                    500: '#00BCF2', // Microsoft Fabric teal
                    600: '#2563eb',
                    700: '#1d4ed8',
                    800: '#1e40af',
                    900: '#1e3a8a',
                },
                success: '#10b981',
                warning: '#f59e0b',
                error: '#ef4444',
                running: '#3b82f6',
                completed: '#10b981',
                failed: '#ef4444',
            },
        },
    },
    plugins: [],
}