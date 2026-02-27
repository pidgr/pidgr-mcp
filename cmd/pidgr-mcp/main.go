// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"github.com/pidgr/pidgr-mcp/internal/auth"
	"github.com/pidgr/pidgr-mcp/internal/tools"
	"github.com/pidgr/pidgr-mcp/internal/transport"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		log.Fatalf("pidgr-mcp: %v", err)
	}
}

func run() error {
	// Parse configuration from environment.
	cfg, err := parseConfig()
	if err != nil {
		return err
	}

	// Create MCP server.
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "pidgr",
		Version: version,
	}, nil)

	// Create clients and register tools based on transport mode.
	switch cfg.Transport {
	case "stdio":
		clients := transport.NewStaticTokenClients(cfg.ApiURL, cfg.apiKey)
		tools.RegisterAll(server, clients)
		return runStdio(server)

	case "http":
		if !strings.HasPrefix(cfg.ApiURL, "https://") {
			slog.Warn("PIDGR_API_URL is not HTTPS — traffic to the backend is unencrypted", "url", cfg.ApiURL)
		}
		clients := transport.NewDynamicTokenClients(cfg.ApiURL)
		tools.RegisterAll(server, clients)
		return runHTTP(server, cfg)

	default:
		return fmt.Errorf("invalid transport %q: must be 'stdio' or 'http'", cfg.Transport)
	}
}

func runStdio(server *mcp.Server) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	return server.Run(ctx, &mcp.StdioTransport{})
}

func runHTTP(server *mcp.Server, cfg *config) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	verifier := auth.NewOIDCVerifier(cfg.AuthPoolID, cfg.AuthRegion, cfg.AuthClientID)

	resourceURL := "https://mcp.pidgr.com"
	metadataURL := resourceURL + "/.well-known/oauth-protected-resource"

	metadata := &oauthex.ProtectedResourceMetadata{
		Resource:               resourceURL,
		AuthorizationServers:   []string{verifier.Issuer()},
		ScopesSupported:        []string{"openid", "profile"},
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "Pidgr MCP Server",
	}

	authMiddleware := mcpauth.RequireBearerToken(verifier.Verify, &mcpauth.RequireBearerTokenOptions{
		ResourceMetadataURL: metadataURL,
	})

	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, nil)

	mux := http.NewServeMux()
	mux.Handle("/.well-known/oauth-protected-resource", mcpauth.ProtectedResourceMetadataHandler(metadata))
	mux.Handle("/", authMiddleware(handler))

	httpServer := &http.Server{
		Addr:           cfg.Addr,
		Handler:        securityHeaders(mux),
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   60 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 8 << 10, // 8 KB
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("HTTP server shutdown error", "error", err)
		}
	}()

	log.Printf("pidgr-mcp: listening on %s (http mode)", cfg.Addr)
	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// securityHeaders adds standard security response headers.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

// config holds parsed environment configuration.
type config struct {
	Transport    string
	ApiURL       string
	apiKey       string
	Addr         string
	AuthPoolID   string
	AuthRegion   string
	AuthClientID string
}

func parseConfig() (*config, error) {
	cfg := &config{
		Transport:    getEnv("PIDGR_MCP_TRANSPORT", "stdio"),
		ApiURL:       getEnv("PIDGR_API_URL", "https://api.pidgr.com"),
		apiKey:       os.Getenv("PIDGR_API_KEY"),
		Addr:         getEnv("PIDGR_MCP_ADDR", ":8080"),
		AuthPoolID:   os.Getenv("PIDGR_AUTH_POOL_ID"),
		AuthRegion:   getEnv("PIDGR_AUTH_REGION", "us-east-1"),
		AuthClientID: os.Getenv("PIDGR_AUTH_CLIENT_ID"),
	}

	switch cfg.Transport {
	case "stdio":
		if cfg.apiKey == "" {
			return nil, fmt.Errorf("PIDGR_API_KEY is required for stdio mode")
		}
	case "http":
		if cfg.AuthPoolID == "" {
			return nil, fmt.Errorf("PIDGR_AUTH_POOL_ID is required for http mode")
		}
	default:
		return nil, fmt.Errorf("PIDGR_MCP_TRANSPORT must be 'stdio' or 'http', got %q", cfg.Transport)
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
