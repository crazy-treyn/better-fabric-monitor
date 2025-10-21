package auth

import (
	"fmt"
	"net/http"
	"strings"
)

// localServer handles the OAuth redirect locally
type localServer struct {
	codeChan  chan string
	errorChan chan error
	server    *http.Server
}

// start starts the local HTTP server
func (ls *localServer) start(redirectURI string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", ls.handleCallback)

	// Use a dynamic port for the callback server
	// Azure CLI uses http://localhost as redirect URI
	port := ":0" // Let OS assign a free port

	// For localhost, try common ports
	if strings.Contains(redirectURI, "localhost") && !strings.Contains(redirectURI, ":") {
		// Try port 8080 first
		port = ":8080"
	} else if strings.Contains(redirectURI, ":") {
		// Extract port from redirect URI
		parts := strings.Split(redirectURI, ":")
		if len(parts) >= 2 {
			portPart := strings.TrimPrefix(parts[len(parts)-1], "/")
			port = ":" + strings.Split(portPart, "/")[0]
		}
	}

	ls.server = &http.Server{
		Addr:    port,
		Handler: mux,
	}

	go func() {
		if err := ls.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ls.errorChan <- fmt.Errorf("server error: %w", err)
		}
	}()
}

// stop stops the local HTTP server
func (ls *localServer) stop() {
	if ls.server != nil {
		ls.server.Close()
	}
}

// handleCallback handles the OAuth callback
func (ls *localServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	errorParam := r.URL.Query().Get("error")
	errorDescription := r.URL.Query().Get("error_description")

	if errorParam != "" {
		ls.errorChan <- fmt.Errorf("OAuth error: %s - %s", errorParam, errorDescription)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Authentication failed: %s", errorDescription)
		return
	}

	if code == "" {
		ls.errorChan <- fmt.Errorf("no authorization code received")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "No authorization code received")
		return
	}

	ls.codeChan <- code

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Authentication Successful</title>
			<style>
				body {
					font-family: Arial, sans-serif;
					text-align: center;
					padding: 50px;
					background-color: #f5f5f5;
				}
				.success {
					color: #28a745;
					font-size: 24px;
					margin-bottom: 20px;
				}
				.message {
					color: #666;
					font-size: 16px;
				}
			</style>
		</head>
		<body>
			<div class="success">✓ Authentication Successful</div>
			<div class="message">You can now close this window and return to the application.</div>
		</body>
		</html>
	`)
}
