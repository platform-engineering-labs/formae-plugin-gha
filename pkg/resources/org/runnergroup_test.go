// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

//go:build integration

package org

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

func TestRunnerGroup_CRUD(t *testing.T) {
	client := testClient(t)
	prov := &runnerGroupProvisioner{client: client, cfg: testConfig(t)}
	ctx := context.Background()
	targetConfig := testTargetConfig(t)
	groupName := uniqueName("formae-inttest-rg")

	// Create
	props, _ := json.Marshal(runnerGroupProperties{
		Name:       groupName,
		Visibility: "all",
	})
	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: RunnerGroupResourceType,
		Properties:   props,
		TargetConfig: targetConfig,
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if createResult.ProgressResult.OperationStatus == resource.OperationStatusFailure {
		if createResult.ProgressResult.ErrorCode == resource.OperationErrorCodeAccessDenied {
			t.Skip("skipping: org admin access required for runner groups")
		}
		t.Fatalf("Create failed: %s", createResult.ProgressResult.StatusMessage)
	}
	nativeID := createResult.ProgressResult.NativeID
	t.Logf("Created runner group %s with nativeID %s", groupName, nativeID)

	defer func() {
		_, _ = prov.Delete(ctx, &resource.DeleteRequest{
			NativeID:     nativeID,
			ResourceType: RunnerGroupResourceType,
			TargetConfig: targetConfig,
		})
	}()

	// Read
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: RunnerGroupResourceType,
		TargetConfig: targetConfig,
	})
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if readResult.Properties == "" {
		t.Fatal("Read returned empty properties")
	}
	var readProps runnerGroupProperties
	json.Unmarshal([]byte(readResult.Properties), &readProps)
	if readProps.Name != groupName {
		t.Errorf("Read name = %q, want %q", readProps.Name, groupName)
	}

	// Update
	updatedName := groupName + "-updated"
	desiredProps, _ := json.Marshal(runnerGroupProperties{
		Name:       updatedName,
		Visibility: "all",
	})
	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:          nativeID,
		ResourceType:      RunnerGroupResourceType,
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
	readResult2, _ := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: RunnerGroupResourceType,
		TargetConfig: targetConfig,
	})
	var readProps2 runnerGroupProperties
	json.Unmarshal([]byte(readResult2.Properties), &readProps2)
	if readProps2.Name != updatedName {
		t.Errorf("After update, name = %q, want %q", readProps2.Name, updatedName)
	}

	// List
	listResult, err := prov.List(ctx, &resource.ListRequest{
		ResourceType: RunnerGroupResourceType,
		TargetConfig: targetConfig,
	})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	found := false
	for _, id := range listResult.NativeIDs {
		if id == nativeID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("List did not contain %s", nativeID)
	}

	// Delete
	deleteResult, err := prov.Delete(ctx, &resource.DeleteRequest{
		NativeID:     nativeID,
		ResourceType: RunnerGroupResourceType,
		TargetConfig: targetConfig,
	})
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if deleteResult.ProgressResult.OperationStatus != resource.OperationStatusSuccess {
		t.Fatalf("Delete failed: %s", deleteResult.ProgressResult.StatusMessage)
	}

	// Read after delete should return not found
	readResult3, _ := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: RunnerGroupResourceType,
		TargetConfig: targetConfig,
	})
	if readResult3.Properties != "" {
		t.Errorf("Read after delete returned properties, expected empty")
	}

	// Delete idempotent
	deleteResult2, _ := prov.Delete(ctx, &resource.DeleteRequest{
		NativeID:     nativeID,
		ResourceType: RunnerGroupResourceType,
		TargetConfig: targetConfig,
	})
	if deleteResult2.ProgressResult.OperationStatus != resource.OperationStatusSuccess {
		t.Error("Delete of already-deleted runner group should succeed (idempotent)")
	}
}
