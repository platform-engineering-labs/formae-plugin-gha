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

func TestPermissions_CRUD(t *testing.T) {
	client := testClient(t)
	prov := &permissionsProvisioner{client: client, cfg: testConfig(t)}
	ctx := context.Background()
	targetConfig := testTargetConfig(t)

	// Read current state first so we can restore it
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     permissionsNativeID,
		ResourceType: PermissionsResourceType,
		TargetConfig: targetConfig,
	})
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if readResult.ErrorCode == resource.OperationErrorCodeAccessDenied {
		t.Skip("skipping: org admin access required for permissions")
	}
	if readResult.Properties == "" {
		t.Fatal("Read returned empty properties")
	}

	var originalProps permissionsProperties
	json.Unmarshal([]byte(readResult.Properties), &originalProps)
	t.Logf("Original permissions: enabledRepositories=%s, allowedActions=%s",
		originalProps.EnabledRepositories, originalProps.AllowedActions)

	// Restore original on cleanup
	defer func() {
		restoreProps, _ := json.Marshal(originalProps)
		_, _ = prov.Update(ctx, &resource.UpdateRequest{
			NativeID:          permissionsNativeID,
			ResourceType:      PermissionsResourceType,
			DesiredProperties: restoreProps,
			TargetConfig:      targetConfig,
		})
	}()

	// Create (set)
	props, _ := json.Marshal(permissionsProperties{
		NativeID:            permissionsNativeID,
		EnabledRepositories: "all",
		AllowedActions:      "all",
	})
	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: PermissionsResourceType,
		Properties:   props,
		TargetConfig: targetConfig,
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if createResult.ProgressResult.OperationStatus == resource.OperationStatusFailure {
		if createResult.ProgressResult.ErrorCode == resource.OperationErrorCodeAccessDenied {
			t.Skip("skipping: org admin access required for permissions")
		}
		t.Fatalf("Create failed: %s", createResult.ProgressResult.StatusMessage)
	}
	t.Log("Set org permissions")

	// Read
	readResult2, _ := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     permissionsNativeID,
		ResourceType: PermissionsResourceType,
		TargetConfig: targetConfig,
	})
	var readProps permissionsProperties
	json.Unmarshal([]byte(readResult2.Properties), &readProps)
	if readProps.EnabledRepositories != "all" {
		t.Errorf("enabledRepositories = %q, want %q", readProps.EnabledRepositories, "all")
	}
	if readProps.AllowedActions != "all" {
		t.Errorf("allowedActions = %q, want %q", readProps.AllowedActions, "all")
	}

	// Update
	desiredProps, _ := json.Marshal(permissionsProperties{
		NativeID:            permissionsNativeID,
		EnabledRepositories: "all",
		AllowedActions:      "local_only",
	})
	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:          permissionsNativeID,
		ResourceType:      PermissionsResourceType,
		DesiredProperties: desiredProps,
		TargetConfig:      targetConfig,
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}
	if updateResult.ProgressResult.OperationStatus != resource.OperationStatusSuccess {
		t.Fatalf("Update failed: %s", updateResult.ProgressResult.StatusMessage)
	}

	// Read after update
	readResult3, _ := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     permissionsNativeID,
		ResourceType: PermissionsResourceType,
		TargetConfig: targetConfig,
	})
	var readProps2 permissionsProperties
	json.Unmarshal([]byte(readResult3.Properties), &readProps2)
	if readProps2.AllowedActions != "local_only" {
		t.Errorf("After update, allowedActions = %q, want %q", readProps2.AllowedActions, "local_only")
	}

	// Delete (reset to defaults)
	deleteResult, _ := prov.Delete(ctx, &resource.DeleteRequest{
		NativeID:     permissionsNativeID,
		ResourceType: PermissionsResourceType,
		TargetConfig: targetConfig,
	})
	if deleteResult.ProgressResult.OperationStatus != resource.OperationStatusSuccess {
		t.Fatalf("Delete failed: %s", deleteResult.ProgressResult.StatusMessage)
	}
}
