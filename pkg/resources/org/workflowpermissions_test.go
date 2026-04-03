// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: Apache-2.0

//go:build integration

package org

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/go-github/v69/github"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

func TestWorkflowPermissions_CRUD(t *testing.T) {
	client := testClient(t)
	prov := &workflowPermissionsProvisioner{client: client, cfg: testConfig(t)}
	ctx := context.Background()
	targetConfig := testTargetConfig(t)

	// Read current state first so we can restore it
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     workflowPermissionsNativeID,
		ResourceType: WorkflowPermissionsResourceType,
		TargetConfig: targetConfig,
	})
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if readResult.ErrorCode == resource.OperationErrorCodeAccessDenied {
		t.Skip("skipping: org admin access required for workflow permissions")
	}
	if readResult.Properties == "" {
		t.Fatal("Read returned empty properties")
	}

	var originalProps workflowPermissionsProperties
	json.Unmarshal([]byte(readResult.Properties), &originalProps)
	t.Logf("Original workflow permissions: default=%s, canApprove=%v",
		originalProps.DefaultWorkflowPermissions, originalProps.CanApprovePullRequestReviews)

	// Restore original on cleanup
	defer func() {
		restoreProps, _ := json.Marshal(originalProps)
		_, _ = prov.Update(ctx, &resource.UpdateRequest{
			NativeID:          workflowPermissionsNativeID,
			ResourceType:      WorkflowPermissionsResourceType,
			DesiredProperties: restoreProps,
			TargetConfig:      targetConfig,
		})
	}()

	// Create (set)
	props, _ := json.Marshal(workflowPermissionsProperties{
		NativeID:                     workflowPermissionsNativeID,
		DefaultWorkflowPermissions:   "read",
		CanApprovePullRequestReviews: github.Ptr(false),
	})
	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: WorkflowPermissionsResourceType,
		Properties:   props,
		TargetConfig: targetConfig,
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if createResult.ProgressResult.OperationStatus == resource.OperationStatusFailure {
		if createResult.ProgressResult.ErrorCode == resource.OperationErrorCodeAccessDenied {
			t.Skip("skipping: org admin access required for workflow permissions")
		}
		t.Fatalf("Create failed: %s", createResult.ProgressResult.StatusMessage)
	}
	t.Log("Set org workflow permissions")

	// Read
	readResult2, _ := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     workflowPermissionsNativeID,
		ResourceType: WorkflowPermissionsResourceType,
		TargetConfig: targetConfig,
	})
	var readProps workflowPermissionsProperties
	json.Unmarshal([]byte(readResult2.Properties), &readProps)
	if readProps.DefaultWorkflowPermissions != "read" {
		t.Errorf("defaultWorkflowPermissions = %q, want %q", readProps.DefaultWorkflowPermissions, "read")
	}

	// Update
	desiredProps, _ := json.Marshal(workflowPermissionsProperties{
		NativeID:                     workflowPermissionsNativeID,
		DefaultWorkflowPermissions:   "write",
		CanApprovePullRequestReviews: github.Ptr(true),
	})
	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:          workflowPermissionsNativeID,
		ResourceType:      WorkflowPermissionsResourceType,
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
		NativeID:     workflowPermissionsNativeID,
		ResourceType: WorkflowPermissionsResourceType,
		TargetConfig: targetConfig,
	})
	var readProps2 workflowPermissionsProperties
	json.Unmarshal([]byte(readResult3.Properties), &readProps2)
	if readProps2.DefaultWorkflowPermissions != "write" {
		t.Errorf("After update, defaultWorkflowPermissions = %q, want %q", readProps2.DefaultWorkflowPermissions, "write")
	}
	if readProps2.CanApprovePullRequestReviews == nil || !*readProps2.CanApprovePullRequestReviews {
		t.Error("After update, canApprovePullRequestReviews should be true")
	}

	// Delete (reset to defaults)
	deleteResult, _ := prov.Delete(ctx, &resource.DeleteRequest{
		NativeID:     workflowPermissionsNativeID,
		ResourceType: WorkflowPermissionsResourceType,
		TargetConfig: targetConfig,
	})
	if deleteResult.ProgressResult.OperationStatus != resource.OperationStatusSuccess {
		t.Fatalf("Delete failed: %s", deleteResult.ProgressResult.StatusMessage)
	}
}
