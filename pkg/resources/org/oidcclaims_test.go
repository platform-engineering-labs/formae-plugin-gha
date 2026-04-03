// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: Apache-2.0

//go:build integration

package org

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

func TestOrgOIDCClaims_CRUD(t *testing.T) {
	client := testClient(t)
	prov := &orgOIDCClaimsProvisioner{client: client, cfg: testConfig(t)}
	ctx := context.Background()
	targetConfig := testTargetConfig(t)

	// Create (set template)
	props, _ := json.Marshal(orgOIDCClaimsProperties{
		NativeID:         orgOIDCClaimsNativeID,
		IncludeClaimKeys: []string{"repo", "context"},
	})
	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: OrgOIDCClaimsResourceType,
		Properties:   props,
		TargetConfig: targetConfig,
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if createResult.ProgressResult.OperationStatus == resource.OperationStatusFailure {
		if createResult.ProgressResult.ErrorCode == resource.OperationErrorCodeAccessDenied {
			t.Skip("skipping: org admin access required for OIDC claims")
		}
		t.Fatalf("Create failed: %s", createResult.ProgressResult.StatusMessage)
	}
	nativeID := createResult.ProgressResult.NativeID
	t.Logf("Set org OIDC claims with nativeID %s", nativeID)

	defer func() {
		_, _ = prov.Delete(ctx, &resource.DeleteRequest{
			NativeID:     nativeID,
			ResourceType: OrgOIDCClaimsResourceType,
			TargetConfig: targetConfig,
		})
	}()

	// Read
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: OrgOIDCClaimsResourceType,
		TargetConfig: targetConfig,
	})
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if readResult.Properties == "" {
		t.Fatal("Read returned empty properties")
	}
	var readProps orgOIDCClaimsProperties
	json.Unmarshal([]byte(readResult.Properties), &readProps)
	if len(readProps.IncludeClaimKeys) < 2 {
		t.Errorf("Read includeClaimKeys length = %d, want >= 2", len(readProps.IncludeClaimKeys))
	}

	// Update
	desiredProps, _ := json.Marshal(orgOIDCClaimsProperties{
		NativeID:         orgOIDCClaimsNativeID,
		IncludeClaimKeys: []string{"repo", "context", "ref"},
	})
	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:          nativeID,
		ResourceType:      OrgOIDCClaimsResourceType,
		DesiredProperties: desiredProps,
		TargetConfig:      targetConfig,
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}
	if updateResult.ProgressResult.OperationStatus != resource.OperationStatusSuccess {
		t.Fatalf("Update failed: %s", updateResult.ProgressResult.StatusMessage)
	}

	// Delete (reset)
	deleteResult, _ := prov.Delete(ctx, &resource.DeleteRequest{
		NativeID:     nativeID,
		ResourceType: OrgOIDCClaimsResourceType,
		TargetConfig: targetConfig,
	})
	if deleteResult.ProgressResult.OperationStatus != resource.OperationStatusSuccess {
		t.Fatalf("Delete failed: %s", deleteResult.ProgressResult.StatusMessage)
	}
}
