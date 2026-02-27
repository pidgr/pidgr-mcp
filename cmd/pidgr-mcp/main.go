// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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
		clients := transport.NewStaticTokenClients(cfg.ApiURL, cfg.ApiKey)
		tools.RegisterAll(server, clients)
		return runStdio(server)

	case "http":
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

	verifier := auth.NewCognitoVerifier(cfg.CognitoPoolID, cfg.CognitoRegion)

	resourceURL := fmt.Sprintf("https://mcp.pidgr.com")
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
		Addr:    cfg.Addr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		_ = httpServer.Close()
	}()

	log.Printf("pidgr-mcp: listening on %s (http mode)", cfg.Addr)
	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// config holds parsed environment configuration.
type config struct {
	Transport      string
	ApiURL         string
	ApiKey         string
	Addr           string
	CognitoPoolID  string
	CognitoRegion  string
}

func parseConfig() (*config, error) {
	cfg := &config{
		Transport:     getEnv("PIDGR_MCP_TRANSPORT", "stdio"),
		ApiURL:        getEnv("PIDGR_API_URL", "https://api.pidgr.com"),
		ApiKey:        os.Getenv("PIDGR_API_KEY"),
		Addr:          getEnv("PIDGR_MCP_ADDR", ":8080"),
		CognitoPoolID: os.Getenv("PIDGR_COGNITO_POOL_ID"),
		CognitoRegion: getEnv("PIDGR_COGNITO_REGION", "us-east-1"),
	}

	switch cfg.Transport {
	case "stdio":
		if cfg.ApiKey == "" {
			return nil, fmt.Errorf("PIDGR_API_KEY is required for stdio mode")
		}
	case "http":
		if cfg.CognitoPoolID == "" {
			return nil, fmt.Errorf("PIDGR_COGNITO_POOL_ID is required for http mode")
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
