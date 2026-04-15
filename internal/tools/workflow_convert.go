// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package tools

import (
	"encoding/json"
	"fmt"

	pidgrv1 "github.com/pidgr/pidgr-proto/gen/go/pidgr/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// workflowFromJSON converts a raw JSON workflow definition to a proto
// WorkflowDefinition. This handles the proto oneof Config field which
// encoding/json cannot unmarshal directly — protojson is required.
func workflowFromJSON(raw json.RawMessage) (*pidgrv1.WorkflowDefinition, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var wf pidgrv1.WorkflowDefinition
	if err := protojson.Unmarshal(raw, &wf); err != nil {
		return nil, fmt.Errorf("unmarshal workflow: %w", err)
	}
	return &wf, nil
}
