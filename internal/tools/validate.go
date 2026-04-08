// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package tools

import "fmt"

const (
	maxPageSize     int32 = 100
	defaultPageSize int32 = 20
	maxBatchSize          = 1000
)

// validDataGovernanceRegions enumerates allowed region codes for data governance.
// Empty string is valid (means "use org default").
var validDataGovernanceRegions = map[string]bool{
	"":      true,
	"EU":    true,
	"LATAM": true,
	"BR":    true,
	"APAC":  true,
	"US":    true,
}

// validateDataGovernanceRegion returns an error if region is not one of the
// allowed values (EU, LATAM, BR, APAC, US, or empty).
func validateDataGovernanceRegion(region string) error {
	if !validDataGovernanceRegions[region] {
		return fmt.Errorf("invalid data_governance_region %q: must be one of EU, LATAM, BR, APAC, US, or empty", region)
	}
	return nil
}

// clampPageSize caps page_size at maxPageSize and defaults to defaultPageSize.
func clampPageSize(size int32) int32 {
	if size <= 0 {
		return defaultPageSize
	}
	if size > maxPageSize {
		return maxPageSize
	}
	return size
}

// validateBatchSize returns an error if the slice exceeds the given limit.
func validateBatchSize(ids []string, max int) error {
	if len(ids) > max {
		return fmt.Errorf("batch size %d exceeds maximum of %d", len(ids), max)
	}
	return nil
}
