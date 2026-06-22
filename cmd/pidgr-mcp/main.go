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
	"github.com/pidgr/pidgr-mcp/internal/oauth"
	"github.com/pidgr/pidgr-mcp/internal/observability"
	"github.com/pidgr/pidgr-mcp/internal/tools"
	"github.com/pidgr/pidgr-mcp/internal/transport"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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

	// Initialize OTEL observability (traces + logs via OTLP, or no-op).
	ctx := context.Background()
	tp, err := observability.InitTracer(ctx, cfg.OTELEndpoint, "pidgr-mcp")
	if err != nil {
		return fmt.Errorf("init tracer: %w", err)
	}
	defer func() { _ = tp.Shutdown(ctx) }()

	lp, err := observability.InitLogger(ctx, cfg.OTELEndpoint, "pidgr-mcp")
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer func() { _ = lp.Shutdown(ctx) }()

	// Fan out slog to both stdout (container logs) and OTEL (remote backend).
	otelHandler := otelslog.NewHandler("pidgr-mcp", otelslog.WithLoggerProvider(lp))
	stdoutHandler := slog.NewJSONHandler(os.Stdout, nil)
	slog.SetDefault(slog.New(observability.NewFanoutHandler(stdoutHandler, otelHandler)))

	// Create MCP server.
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "pidgr",
		Version: version,
	}, nil)

	// Create clients and register tools based on transport mode.
	switch cfg.Transport {
	case "stdio":
		clients, err := newStdioClients(cfg)
		if err != nil {
			return err
		}
		tools.RegisterAll(server, clients)
		return runStdio(server)

	case "http":
		if !strings.HasPrefix(cfg.ApiURL, "https://") {
			slog.Warn("PIDGR_API_URL is not HTTPS — traffic to the backend is unencrypted", "url", cfg.ApiURL)
		}
		if !strings.HasPrefix(cfg.IntegrationsURL, "https://") {
			slog.Warn("PIDGR_INTEGRATIONS_URL is not HTTPS — IntegrationsService traffic is unencrypted", "url", cfg.IntegrationsURL)
		}
		clients := transport.NewDynamicTokenClientsWithIntegrationsURL(cfg.ApiURL, cfg.IntegrationsURL)
		tools.RegisterAll(server, clients)
		return runHTTP(server, cfg)

	default:
		return fmt.Errorf("invalid transport %q: must be 'stdio' or 'http'", cfg.Transport)
	}
}

// newStdioClients builds the stdio authentication path. stdio is OAuth-only: it
// always uses the OAuth authorization_code + PKCE browser flow and resolves
// tokens lazily on the first RPC.
func newStdioClients(cfg *config) (*transport.Clients, error) {
	slog.Info("stdio auth: using OAuth (authorization_code + PKCE)", "issuer", cfg.OAuthIssuer)
	oauthClient, err := oauth.NewClient(oauth.Config{
		Issuer:   cfg.OAuthIssuer,
		ClientID: cfg.OAuthClientID,
		Scope:    cfg.OAuthScope,
	})
	if err != nil {
		return nil, fmt.Errorf("init oauth client: %w", err)
	}
	return transport.NewOAuthClientsWithIntegrationsURL(cfg.ApiURL, cfg.IntegrationsURL, oauthClient.AccessToken), nil
}

func runStdio(server *mcp.Server) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	return server.Run(ctx, &mcp.StdioTransport{})
}

func runHTTP(server *mcp.Server, cfg *config) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// HTTP mode is a hosted MCP acting as an OAuth resource server. The bearer
	// must be a Pidgr-issued OAuth JWT (RS256), verified against the provider's
	// JWKS. pidgr_k_ API keys are no longer accepted. On success the same bearer
	// is forwarded to pidgr-api, which does the authoritative scope∩principal
	// enforcement (A6).
	verifier := oauth.NewVerifier(oauth.VerifierConfig{Issuer: cfg.OAuthIssuer})
	authMiddleware := mcpauth.RequireBearerToken(verifier.Verify, &mcpauth.RequireBearerTokenOptions{
		ResourceMetadataURL: cfg.OAuthIssuer + resourceMetadataPath,
	})

	prmHandler := mcpauth.ProtectedResourceMetadataHandler(protectedResourceMetadata(cfg))

	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, nil)

	mux := newHTTPMux(authMiddleware, handler, prmHandler)

	httpServer := &http.Server{
		Addr:           cfg.Addr,
		Handler:        otelhttp.NewHandler(securityHeaders(mux), "pidgr-mcp"),
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

// newHTTPMux builds the HTTP mux used by the server. It registers three routes:
//
//   - GET /healthz returns 200 with a JSON body. It is unauthenticated so that
//     load-balancer and container orchestrator health checks can probe the
//     server without credentials.
//   - GET /.well-known/oauth-protected-resource serves RFC 9728 protected-resource
//     metadata advertising the Pidgr OAuth provider as the authorization server.
//     It is unauthenticated so MCP clients can discover where to authenticate.
//   - / is the catch-all for the MCP transport and is wrapped in the bearer
//     token middleware so every other path requires a valid OAuth JWT.
//
// http.ServeMux matches the more specific pattern first, so /healthz and the
// well-known path take precedence over the / catch-all.
//
// This constructor exists so tests exercise the real routing instead of a
// hand-rolled duplicate — registering routes here instead of inline in runHTTP
// prevents a route from being silently dropped again.
func newHTTPMux(authMiddleware func(http.Handler) http.Handler, mcpHandler, prmHandler http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.Handle(resourceMetadataPath, prmHandler)
	mux.Handle("/", authMiddleware(mcpHandler))
	return mux
}

const resourceMetadataPath = "/.well-known/oauth-protected-resource"

// protectedResourceMetadata builds the RFC 9728 metadata advertising the Pidgr
// OAuth provider as this resource server's authorization server.
func protectedResourceMetadata(cfg *config) *oauthex.ProtectedResourceMetadata {
	return &oauthex.ProtectedResourceMetadata{
		Resource:               cfg.ResourceURL,
		AuthorizationServers:   []string{cfg.OAuthIssuer},
		ScopesSupported:        strings.Fields(cfg.OAuthScope),
		BearerMethodsSupported: []string{"header"},
	}
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
	Transport       string
	ApiURL          string
	IntegrationsURL string
	Addr            string
	OTELEndpoint    string
	OAuthIssuer     string
	OAuthClientID   string
	OAuthScope      string
	ResourceURL     string
}

const (
	defaultOAuthIssuer   = "https://auth.pidgr.com"
	defaultOAuthClientID = "pidgr-mcp"
	defaultOAuthScope    = "campaigns:read campaigns:write templates:write channels:dispatch reachability:write members:read"
	defaultResourceURL   = "https://mcp.pidgr.com"
)

func parseConfig() (*config, error) {
	apiURL := getEnv("PIDGR_API_URL", "https://api.pidgr.com")
	cfg := &config{
		Transport: getEnv("PIDGR_MCP_TRANSPORT", "stdio"),
		ApiURL:    apiURL,
		// PIDGR_INTEGRATIONS_URL points at the IntegrationsService endpoint.
		// Falls back to PIDGR_API_URL when unset — useful when the
		// IntegrationsService is co-hosted at the same gRPC ingress as the
		// main API.
		IntegrationsURL: getEnv("PIDGR_INTEGRATIONS_URL", apiURL),
		Addr:            getEnv("PIDGR_MCP_ADDR", ":8080"),
		OTELEndpoint:    os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		OAuthIssuer:     getEnv("PIDGR_OAUTH_ISSUER", defaultOAuthIssuer),
		OAuthClientID:   getEnv("PIDGR_OAUTH_CLIENT_ID", defaultOAuthClientID),
		OAuthScope:      getEnv("PIDGR_OAUTH_SCOPE", defaultOAuthScope),
		// ResourceURL is this resource server's identifier, advertised in the
		// protected-resource metadata (HTTP mode).
		ResourceURL: getEnv("PIDGR_MCP_RESOURCE_URL", defaultResourceURL),
	}

	switch cfg.Transport {
	case "stdio":
		// stdio is OAuth-only; the browser flow needs no up-front validation here.
	case "http":
		// HTTP mode verifies OAuth Bearer JWTs via the Authorization header.
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
