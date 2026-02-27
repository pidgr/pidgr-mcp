// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package auth

import (
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

// NewProtectedResourceMetadata builds the OAuth 2.0 Protected Resource Metadata
// for the MCP server (RFC 9728).
func NewProtectedResourceMetadata(resourceURL, cognitoIssuer string) *oauthex.ProtectedResourceMetadata {
	return &oauthex.ProtectedResourceMetadata{
		Resource:               resourceURL,
		AuthorizationServers:   []string{cognitoIssuer},
		ScopesSupported:        []string{"openid", "profile"},
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "Pidgr MCP Server",
	}
}
