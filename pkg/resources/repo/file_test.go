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

func TestRepoFile_CRUD(t *testing.T) {
	client := testClient(t)
	prov := &fileProvisioner{client: client, cfg: testConfig(t)}
	ctx := context.Background()
	targetConfig := testTargetConfig(t)
	filePath := "test/" + uniqueName("inttest") + ".txt"

	// Create
	props, _ := json.Marshal(fileProperties{
		Path:          filePath,
		Content:       "hello from integration test",
		CommitMessage: "test: create integration test file",
	})
	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: FileResourceType,
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
	t.Logf("Created file %s", nativeID)

	var createProps fileProperties
	json.Unmarshal(createResult.ProgressResult.ResourceProperties, &createProps)
	if createProps.SHA == "" {
		t.Error("Create result missing SHA")
	}

	defer func() {
		_, _ = prov.Delete(ctx, &resource.DeleteRequest{
			NativeID:     nativeID,
			ResourceType: FileResourceType,
			TargetConfig: targetConfig,
		})
	}()

	// Read
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: FileResourceType,
		TargetConfig: targetConfig,
	})
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	var readProps fileProperties
	json.Unmarshal([]byte(readResult.Properties), &readProps)
	if readProps.Content != "hello from integration test" {
		t.Errorf("Read content = %q, want %q", readProps.Content, "hello from integration test")
	}
	if readProps.SHA == "" {
		t.Error("Read result missing SHA")
	}

	// Update
	desiredProps, _ := json.Marshal(fileProperties{
		Path:          filePath,
		Content:       "updated content",
		CommitMessage: "test: update integration test file",
	})
	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:          nativeID,
		ResourceType:      FileResourceType,
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
		ResourceType: FileResourceType,
		TargetConfig: targetConfig,
	})
	var readProps2 fileProperties
	json.Unmarshal([]byte(readResult2.Properties), &readProps2)
	if readProps2.Content != "updated content" {
		t.Errorf("After update content = %q, want %q", readProps2.Content, "updated content")
	}

	// Delete
	deleteResult, _ := prov.Delete(ctx, &resource.DeleteRequest{
		NativeID:     nativeID,
		ResourceType: FileResourceType,
		TargetConfig: targetConfig,
	})
	if deleteResult.ProgressResult.OperationStatus != resource.OperationStatusSuccess {
		t.Fatalf("Delete failed: %s", deleteResult.ProgressResult.StatusMessage)
	}

	// Read after delete
	readResult3, _ := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: FileResourceType,
		TargetConfig: targetConfig,
	})
	if readResult3.Properties != "" {
		t.Error("Read after delete should return empty")
	}
}
