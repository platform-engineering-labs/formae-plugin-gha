// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

//go:build integration

package env

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

func TestEnvironment_CRUD(t *testing.T) {
	client := testClient(t)
	cfg := testConfig(t)
	prov := &environmentProvisioner{client: client, cfg: cfg}
	ctx := context.Background()
	targetConfig := testTargetConfig(t)
	envName := uniqueName("formae-inttest-env")

	// Create
	props, _ := json.Marshal(environmentProperties{
		Name:      envName,
		WaitTimer: intPtr(0),
	})
	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: EnvironmentResourceType,
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
	t.Logf("Created environment %s", nativeID)

	defer func() {
		_, _ = prov.Delete(ctx, &resource.DeleteRequest{
			NativeID:     nativeID,
			ResourceType: EnvironmentResourceType,
			TargetConfig: targetConfig,
		})
	}()

	// Read
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: EnvironmentResourceType,
		TargetConfig: targetConfig,
	})
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	var readProps environmentProperties
	json.Unmarshal([]byte(readResult.Properties), &readProps)
	if readProps.Name != envName {
		t.Errorf("Read name = %q, want %q", readProps.Name, envName)
	}

	// Update — add wait timer
	desiredProps, _ := json.Marshal(environmentProperties{
		Name:      envName,
		WaitTimer: intPtr(5),
	})
	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:          nativeID,
		ResourceType:      EnvironmentResourceType,
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
		ResourceType: EnvironmentResourceType,
		TargetConfig: targetConfig,
	})
	var readProps2 environmentProperties
	json.Unmarshal([]byte(readResult2.Properties), &readProps2)
	if readProps2.WaitTimer == nil || *readProps2.WaitTimer != 5 {
		t.Errorf("After update, waitTimer = %v, want 5", readProps2.WaitTimer)
	}

	// List
	listResult, _ := prov.List(ctx, &resource.ListRequest{
		ResourceType: EnvironmentResourceType,
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
		ResourceType: EnvironmentResourceType,
		TargetConfig: targetConfig,
	})
	if deleteResult.ProgressResult.OperationStatus != resource.OperationStatusSuccess {
		t.Fatalf("Delete failed: %s", deleteResult.ProgressResult.StatusMessage)
	}
}

func TestEnvVariable_CRUD(t *testing.T) {
	client := testClient(t)
	cfg := testConfig(t)
	owner, _ := testOwnerRepo(t)
	ctx := context.Background()
	targetConfig := testTargetConfig(t)
	envName := uniqueName("formae-inttest-envvar")

	// Create environment first
	envProv := &environmentProvisioner{client: client, cfg: cfg}
	envProps, _ := json.Marshal(environmentProperties{Name: envName})
	envResult, _ := envProv.Create(ctx, &resource.CreateRequest{
		ResourceType: EnvironmentResourceType,
		Properties:   envProps,
		TargetConfig: targetConfig,
	})
	defer func() {
		_, _ = envProv.Delete(ctx, &resource.DeleteRequest{
			NativeID:     envResult.ProgressResult.NativeID,
			ResourceType: EnvironmentResourceType,
			TargetConfig: targetConfig,
		})
	}()

	// Test env variable CRUD
	varProv := &envVariableProvisioner{client: client, cfg: cfg}
	varName := uniqueName("FORMAE_INTTEST_ENVVAR")

	props, _ := json.Marshal(envVariableProperties{Environment: envName, Name: varName, Value: "env-val-1"})
	createResult, err := varProv.Create(ctx, &resource.CreateRequest{
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
	t.Logf("Created env variable %s", nativeID)
	_ = owner

	defer func() {
		_, _ = varProv.Delete(ctx, &resource.DeleteRequest{
			NativeID:     nativeID,
			ResourceType: VariableResourceType,
			TargetConfig: targetConfig,
		})
	}()

	// Read
	readResult, _ := varProv.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: VariableResourceType,
		TargetConfig: targetConfig,
	})
	var readProps envVariableProperties
	json.Unmarshal([]byte(readResult.Properties), &readProps)
	if readProps.Value != "env-val-1" {
		t.Errorf("Read value = %q, want %q", readProps.Value, "env-val-1")
	}

	// Update
	desired, _ := json.Marshal(envVariableProperties{Environment: envName, Name: varName, Value: "env-val-2"})
	updateResult, _ := varProv.Update(ctx, &resource.UpdateRequest{
		NativeID:          nativeID,
		ResourceType:      VariableResourceType,
		DesiredProperties: desired,
		TargetConfig:      targetConfig,
	})
	if updateResult.ProgressResult.OperationStatus != resource.OperationStatusSuccess {
		t.Fatalf("Update failed: %s", updateResult.ProgressResult.StatusMessage)
	}

	// Delete
	deleteResult, _ := varProv.Delete(ctx, &resource.DeleteRequest{
		NativeID:     nativeID,
		ResourceType: VariableResourceType,
		TargetConfig: targetConfig,
	})
	if deleteResult.ProgressResult.OperationStatus != resource.OperationStatusSuccess {
		t.Fatalf("Delete failed: %s", deleteResult.ProgressResult.StatusMessage)
	}
}

func TestEnvSecret_CRUD(t *testing.T) {
	client := testClient(t)
	cfg := testConfig(t)
	ctx := context.Background()
	targetConfig := testTargetConfig(t)
	envName := uniqueName("formae-inttest-envsec")

	// Create environment first
	envProv := &environmentProvisioner{client: client, cfg: cfg}
	envProps, _ := json.Marshal(environmentProperties{Name: envName})
	envResult, _ := envProv.Create(ctx, &resource.CreateRequest{
		ResourceType: EnvironmentResourceType,
		Properties:   envProps,
		TargetConfig: targetConfig,
	})
	defer func() {
		_, _ = envProv.Delete(ctx, &resource.DeleteRequest{
			NativeID:     envResult.ProgressResult.NativeID,
			ResourceType: EnvironmentResourceType,
			TargetConfig: targetConfig,
		})
	}()

	// Test env secret CRUD
	secProv := &envSecretProvisioner{client: client, cfg: cfg}
	secName := uniqueName("FORMAE_INTTEST_ENVSEC")

	props, _ := json.Marshal(envSecretProperties{Environment: envName, Name: secName, Value: "env-secret-1"})
	createResult, err := secProv.Create(ctx, &resource.CreateRequest{
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
	t.Logf("Created env secret %s", nativeID)

	defer func() {
		_, _ = secProv.Delete(ctx, &resource.DeleteRequest{
			NativeID:     nativeID,
			ResourceType: SecretResourceType,
			TargetConfig: targetConfig,
		})
	}()

	// Read
	readResult, _ := secProv.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: SecretResourceType,
		TargetConfig: targetConfig,
	})
	if readResult.Properties == "" {
		t.Fatal("Read returned empty")
	}

	// Delete
	deleteResult, _ := secProv.Delete(ctx, &resource.DeleteRequest{
		NativeID:     nativeID,
		ResourceType: SecretResourceType,
		TargetConfig: targetConfig,
	})
	if deleteResult.ProgressResult.OperationStatus != resource.OperationStatusSuccess {
		t.Fatalf("Delete failed: %s", deleteResult.ProgressResult.StatusMessage)
	}
}

func intPtr(i int) *int { return &i }
