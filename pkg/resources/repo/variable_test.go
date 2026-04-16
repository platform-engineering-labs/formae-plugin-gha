// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

//go:build integration

package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/go-github/v69/github"
	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/config"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

func testClient(t *testing.T) *github.Client {
	t.Helper()
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("GITHUB_TOKEN not set")
	}
	return github.NewClient(nil).WithAuthToken(token)
}

func testOwnerRepo(t *testing.T) (string, string) {
	t.Helper()
	owner := os.Getenv("GHA_TEST_OWNER")
	repo := os.Getenv("GHA_TEST_REPO")
	if owner == "" || repo == "" {
		t.Skip("GHA_TEST_OWNER or GHA_TEST_REPO not set")
	}
	return owner, repo
}

func testTargetConfig(t *testing.T) []byte {
	t.Helper()
	owner, repo := testOwnerRepo(t)
	cfg, _ := json.Marshal(map[string]string{"Owner": owner, "Repo": repo})
	return cfg
}

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	owner, repo := testOwnerRepo(t)
	return &config.Config{Owner: owner, Repo: repo}
}

func uniqueName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano()%100000)
}

func TestRepoVariable_CRUD(t *testing.T) {
	client := testClient(t)
	prov := &variableProvisioner{client: client, cfg: testConfig(t)}
	ctx := context.Background()
	targetConfig := testTargetConfig(t)
	varName := uniqueName("FORMAE_INTTEST_VAR")

	// Create
	props, _ := json.Marshal(variableProperties{Name: varName, Value: "initial-value"})
	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: VariableResourceType,
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
	t.Logf("Created variable %s with nativeID %s", varName, nativeID)

	// Cleanup
	defer func() {
		_, _ = prov.Delete(ctx, &resource.DeleteRequest{
			NativeID:     nativeID,
			ResourceType: VariableResourceType,
			TargetConfig: targetConfig,
		})
	}()

	// Read
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: VariableResourceType,
		TargetConfig: targetConfig,
	})
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if readResult.Properties == "" {
		t.Fatal("Read returned empty properties")
	}
	var readProps variableProperties
	json.Unmarshal([]byte(readResult.Properties), &readProps)
	if readProps.Value != "initial-value" {
		t.Errorf("Read value = %q, want %q", readProps.Value, "initial-value")
	}

	// Update
	desiredProps, _ := json.Marshal(variableProperties{Name: varName, Value: "updated-value"})
	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:           nativeID,
		ResourceType:       VariableResourceType,
		DesiredProperties:  desiredProps,
		TargetConfig:       targetConfig,
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}
	if updateResult.ProgressResult.OperationStatus != resource.OperationStatusSuccess {
		t.Fatalf("Update failed: %s", updateResult.ProgressResult.StatusMessage)
	}

	// Read again to verify update
	readResult2, _ := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: VariableResourceType,
		TargetConfig: targetConfig,
	})
	var readProps2 variableProperties
	json.Unmarshal([]byte(readResult2.Properties), &readProps2)
	if readProps2.Value != "updated-value" {
		t.Errorf("After update, value = %q, want %q", readProps2.Value, "updated-value")
	}

	// List
	listResult, err := prov.List(ctx, &resource.ListRequest{
		ResourceType: VariableResourceType,
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
		ResourceType: VariableResourceType,
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
		ResourceType: VariableResourceType,
		TargetConfig: targetConfig,
	})
	if readResult3.Properties != "" {
		t.Errorf("Read after delete returned properties, expected empty")
	}

	// Delete idempotent
	deleteResult2, _ := prov.Delete(ctx, &resource.DeleteRequest{
		NativeID:     nativeID,
		ResourceType: VariableResourceType,
		TargetConfig: targetConfig,
	})
	if deleteResult2.ProgressResult.OperationStatus != resource.OperationStatusSuccess {
		t.Error("Delete of already-deleted variable should succeed (idempotent)")
	}
}
