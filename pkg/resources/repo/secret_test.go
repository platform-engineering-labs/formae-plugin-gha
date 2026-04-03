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

func TestRepoSecret_CRUD(t *testing.T) {
	client := testClient(t)
	prov := &secretProvisioner{client: client, cfg: testConfig(t)}
	ctx := context.Background()
	targetConfig := testTargetConfig(t)
	secretName := uniqueName("FORMAE_INTTEST_SECRET")

	// Create
	props, _ := json.Marshal(secretProperties{Name: secretName, Value: "s3cret-value-1"})
	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: SecretResourceType,
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
	t.Logf("Created secret %s", nativeID)

	// Check valueHash is set
	var createProps secretProperties
	json.Unmarshal(createResult.ProgressResult.ResourceProperties, &createProps)
	if createProps.ValueHash == "" {
		t.Error("Create result missing valueHash")
	}

	defer func() {
		_, _ = prov.Delete(ctx, &resource.DeleteRequest{
			NativeID:     nativeID,
			ResourceType: SecretResourceType,
			TargetConfig: targetConfig,
		})
	}()

	// Read — should return name but no value (write-only)
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: SecretResourceType,
		TargetConfig: targetConfig,
	})
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if readResult.Properties == "" {
		t.Fatal("Read returned empty properties")
	}
	var readProps secretProperties
	json.Unmarshal([]byte(readResult.Properties), &readProps)
	if readProps.Name != secretName {
		t.Errorf("Read name = %q, want %q", readProps.Name, secretName)
	}
	if readProps.Value != "" {
		t.Error("Read should not return secret value")
	}

	// Update
	desiredProps, _ := json.Marshal(secretProperties{Name: secretName, Value: "s3cret-value-2"})
	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:          nativeID,
		ResourceType:      SecretResourceType,
		DesiredProperties: desiredProps,
		TargetConfig:      targetConfig,
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}
	if updateResult.ProgressResult.OperationStatus != resource.OperationStatusSuccess {
		t.Fatalf("Update failed: %s", updateResult.ProgressResult.StatusMessage)
	}
	// Verify valueHash changed
	var updateProps secretProperties
	json.Unmarshal(updateResult.ProgressResult.ResourceProperties, &updateProps)
	if updateProps.ValueHash == createProps.ValueHash {
		t.Error("ValueHash should differ after update")
	}

	// List
	listResult, _ := prov.List(ctx, &resource.ListRequest{
		ResourceType: SecretResourceType,
		TargetConfig: targetConfig,
	})
	found := false
	for _, id := range listResult.NativeIDs {
		if id == nativeID {
			found = true
		}
	}
	if !found {
		t.Errorf("List did not contain %s", nativeID)
	}

	// Delete
	deleteResult, _ := prov.Delete(ctx, &resource.DeleteRequest{
		NativeID:     nativeID,
		ResourceType: SecretResourceType,
		TargetConfig: targetConfig,
	})
	if deleteResult.ProgressResult.OperationStatus != resource.OperationStatusSuccess {
		t.Fatalf("Delete failed: %s", deleteResult.ProgressResult.StatusMessage)
	}

	// Read after delete
	readResult2, _ := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: SecretResourceType,
		TargetConfig: targetConfig,
	})
	if readResult2.Properties != "" {
		t.Error("Read after delete should return empty properties")
	}
}
