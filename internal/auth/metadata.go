// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package auth

import (
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

// NewProtectedResourceMetadata builds the OAuth 2.0 Protected Resource Metadata
// for the MCP server (RFC 9728). The authorizationServer parameter is the URL
// where clients should fetch the authorization server metadata from — typically
// the resource server itself when using a DCR shim.
func NewProtectedResourceMetadata(resourceURL, authorizationServer string) *oauthex.ProtectedResourceMetadata {
	return &oauthex.ProtectedResourceMetadata{
		Resource:               resourceURL,
		AuthorizationServers:   []string{authorizationServer},
		ScopesSupported:        []string{"openid", "profile"},
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "Pidgr MCP Server",
	}
}
