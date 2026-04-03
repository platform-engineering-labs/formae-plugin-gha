// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: Apache-2.0

//go:build integration

package repo

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

func TestWorkflow_CRUD(t *testing.T) {
	client := testClient(t)
	prov := &workflowProvisioner{client: client, cfg: testConfig(t)}
	ctx := context.Background()
	targetConfig := testTargetConfig(t)
	workflowPath := ".github/workflows/" + uniqueName("formae-inttest") + ".yml"

	// Build structured workflow properties.
	createProps := map[string]interface{}{
		"path": workflowPath,
		"name": "Integration Test Workflow",
		"on": map[string]interface{}{
			"workflow_dispatch": nil,
		},
		"jobs": map[string]interface{}{
			"test": map[string]interface{}{
				"runs-on": "ubuntu-latest",
				"steps": []interface{}{
					map[string]interface{}{
						"name": "Hello",
						"run":  "echo hello",
					},
				},
			},
		},
	}

	props, _ := json.Marshal(createProps)

	// Create
	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: WorkflowResourceType,
		Properties:   props,
		TargetConfig: targetConfig,
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if createResult.ProgressResult.OperationStatus != resource.OperationStatusSuccess {
		t.Fatalf("Create failed: %s", createResult.ProgressResult.StatusMessage)
	}
	nativeID := createResult.ProgressResult.NativeID
	t.Logf("Created workflow %s", nativeID)

	if nativeID != workflowPath {
		t.Errorf("NativeID = %q, want %q", nativeID, workflowPath)
	}

	// Cleanup
	defer func() {
		_, _ = prov.Delete(ctx, &resource.DeleteRequest{
			NativeID:     nativeID,
			ResourceType: WorkflowResourceType,
			TargetConfig: targetConfig,
		})
	}()

	// Read
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: WorkflowResourceType,
		TargetConfig: targetConfig,
	})
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if readResult.Properties == "" {
		t.Fatal("Read returned empty properties")
	}

	var readProps map[string]interface{}
	if err := json.Unmarshal([]byte(readResult.Properties), &readProps); err != nil {
		t.Fatalf("Failed to unmarshal read properties: %v", err)
	}
	if readProps["path"] != workflowPath {
		t.Errorf("Read path = %q, want %q", readProps["path"], workflowPath)
	}
	if readProps["name"] != "Integration Test Workflow" {
		t.Errorf("Read name = %q, want %q", readProps["name"], "Integration Test Workflow")
	}

	// Verify jobs structure is present.
	jobs, ok := readProps["jobs"].(map[string]interface{})
	if !ok {
		t.Fatal("Read jobs is not a map")
	}
	if _, ok := jobs["test"]; !ok {
		t.Error("Read jobs missing 'test' job")
	}

	// Update — change the step.
	updateProps := map[string]interface{}{
		"path": workflowPath,
		"name": "Updated Integration Test Workflow",
		"on": map[string]interface{}{
			"workflow_dispatch": nil,
		},
		"jobs": map[string]interface{}{
			"test": map[string]interface{}{
				"runs-on": "ubuntu-latest",
				"steps": []interface{}{
					map[string]interface{}{
						"name": "Updated Hello",
						"run":  "echo updated",
					},
				},
			},
		},
	}
	desiredProps, _ := json.Marshal(updateProps)
	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:          nativeID,
		ResourceType:      WorkflowResourceType,
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
	readResult2, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: WorkflowResourceType,
		TargetConfig: targetConfig,
	})
	if err != nil {
		t.Fatalf("Read after update error: %v", err)
	}

	var readProps2 map[string]interface{}
	json.Unmarshal([]byte(readResult2.Properties), &readProps2)
	if readProps2["name"] != "Updated Integration Test Workflow" {
		t.Errorf("After update, name = %q, want %q", readProps2["name"], "Updated Integration Test Workflow")
	}

	// Delete
	deleteResult, err := prov.Delete(ctx, &resource.DeleteRequest{
		NativeID:     nativeID,
		ResourceType: WorkflowResourceType,
		TargetConfig: targetConfig,
	})
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if deleteResult.ProgressResult.OperationStatus != resource.OperationStatusSuccess {
		t.Fatalf("Delete failed: %s", deleteResult.ProgressResult.StatusMessage)
	}

	// Read after delete — should be not found.
	readResult3, _ := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: WorkflowResourceType,
		TargetConfig: targetConfig,
	})
	if readResult3.Properties != "" {
		t.Error("Read after delete should return empty properties")
	}
}
