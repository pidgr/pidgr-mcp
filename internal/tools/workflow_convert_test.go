// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package tools

import (
	"encoding/json"
	"testing"

	pidgrv1 "github.com/pidgr/pidgr-proto/gen/go/pidgr/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowFromJSON_SendNotification(t *testing.T) {
	raw := json.RawMessage(`{
		"steps": [{
			"id": "step_1",
			"type": "STEP_TYPE_SEND_NOTIFICATION",
			"sendNotification": {"type": "push", "actionType": "ACTION_TYPE_ACK"},
			"transitions": {"completed": "step_2"}
		}, {
			"id": "step_2",
			"type": "STEP_TYPE_DEADLINE_CHECK",
			"deadlineCheck": {"delay": "5m"},
			"transitions": {"completed": "step_3"}
		}, {
			"id": "step_3",
			"type": "STEP_TYPE_MARK_MISSED",
			"transitions": {}
		}]
	}`)

	wf, err := workflowFromJSON(raw)
	require.NoError(t, err)
	require.Len(t, wf.Steps, 3)

	// Step 1: SendNotification
	assert.Equal(t, "step_1", wf.Steps[0].Id)
	assert.Equal(t, pidgrv1.StepType_STEP_TYPE_SEND_NOTIFICATION, wf.Steps[0].Type)
	assert.NotNil(t, wf.Steps[0].GetSendNotification())
	assert.Equal(t, "push", wf.Steps[0].GetSendNotification().Type)
	assert.Equal(t, pidgrv1.ActionType_ACTION_TYPE_ACK, wf.Steps[0].GetSendNotification().ActionType)
	assert.Equal(t, "step_2", wf.Steps[0].Transitions["completed"])

	// Step 2: DeadlineCheck
	assert.Equal(t, pidgrv1.StepType_STEP_TYPE_DEADLINE_CHECK, wf.Steps[1].Type)
	assert.NotNil(t, wf.Steps[1].GetDeadlineCheck())
	assert.Equal(t, "5m", wf.Steps[1].GetDeadlineCheck().Delay)

	// Step 3: MarkMissed
	assert.Equal(t, pidgrv1.StepType_STEP_TYPE_MARK_MISSED, wf.Steps[2].Type)
}

func TestWorkflowFromJSON_Escalate(t *testing.T) {
	raw := json.RawMessage(`{
		"steps": [{
			"id": "step_1",
			"type": "STEP_TYPE_ESCALATE",
			"escalateConfig": {
				"condition": "ESCALATION_CONDITION_IF_NOT_ACKED",
				"repeatCount": 2,
				"repeatIntervalMinutes": 30
			},
			"transitions": {}
		}]
	}`)

	wf, err := workflowFromJSON(raw)
	require.NoError(t, err)
	require.Len(t, wf.Steps, 1)

	esc := wf.Steps[0].GetEscalateConfig()
	require.NotNil(t, esc)
	assert.Equal(t, pidgrv1.EscalationCondition_ESCALATION_CONDITION_IF_NOT_ACKED, esc.Condition)
	assert.Equal(t, int32(2), esc.RepeatCount)
	assert.Equal(t, int32(30), esc.RepeatIntervalMinutes)
}

func TestWorkflowFromJSON_NilInput(t *testing.T) {
	wf, err := workflowFromJSON(nil)
	assert.NoError(t, err)
	assert.Nil(t, wf)
}

func TestWorkflowFromJSON_InvalidJSON(t *testing.T) {
	raw := json.RawMessage(`{invalid`)
	_, err := workflowFromJSON(raw)
	assert.Error(t, err)
}

func TestWorkflowFromJSON_NumericStepType(t *testing.T) {
	// MCP clients may send numeric step types
	raw := json.RawMessage(`{
		"steps": [{
			"id": "step_1",
			"type": 1,
			"sendNotification": {"type": "push"},
			"transitions": {}
		}]
	}`)

	wf, err := workflowFromJSON(raw)
	require.NoError(t, err)
	assert.Equal(t, pidgrv1.StepType_STEP_TYPE_SEND_NOTIFICATION, wf.Steps[0].Type)
	assert.NotNil(t, wf.Steps[0].GetSendNotification())
}
