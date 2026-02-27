// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package tools

import (
	"testing"
)

func TestClampPageSize(t *testing.T) {
	tests := []struct {
		name string
		in   int32
		want int32
	}{
		{"zero defaults", 0, defaultPageSize},
		{"negative defaults", -1, defaultPageSize},
		{"within range", 50, 50},
		{"at max", maxPageSize, maxPageSize},
		{"over max capped", 200, maxPageSize},
		{"one", 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampPageSize(tt.in)
			if got != tt.want {
				t.Errorf("clampPageSize(%d) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestValidateBatchSize(t *testing.T) {
	t.Run("within limit", func(t *testing.T) {
		ids := make([]string, 50)
		if err := validateBatchSize(ids, 100); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("at limit", func(t *testing.T) {
		ids := make([]string, 100)
		if err := validateBatchSize(ids, 100); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("over limit", func(t *testing.T) {
		ids := make([]string, 101)
		err := validateBatchSize(ids, 100)
		if err == nil {
			t.Fatal("expected error for oversized batch")
		}
	})

	t.Run("empty", func(t *testing.T) {
		if err := validateBatchSize(nil, 100); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
