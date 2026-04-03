// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: Apache-2.0

//go:build integration

package repo

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/go-github/v69/github"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

func TestRepoOIDCClaims_CRUD(t *testing.T) {
	client := testClient(t)
	prov := &repoOIDCClaimsProvisioner{client: client, cfg: testConfig(t)}
	ctx := context.Background()
	targetConfig := testTargetConfig(t)

	// Create (set template)
	props, _ := json.Marshal(repoOIDCClaimsProperties{
		NativeID:         oidcClaimsNativeID,
		UseDefault:       github.Ptr(false),
		IncludeClaimKeys: []string{"repo", "context"},
	})
	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: OIDCClaimsResourceType,
		Properties:   props,
		TargetConfig: targetConfig,
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if createResult.ProgressResult.OperationStatus == resource.OperationStatusFailure {
		if createResult.ProgressResult.ErrorCode == resource.OperationErrorCodeAccessDenied {
			t.Skip("skipping: insufficient permissions for OIDC claims")
		}
		t.Fatalf("Create failed: %s", createResult.ProgressResult.StatusMessage)
	}
	nativeID := createResult.ProgressResult.NativeID
	t.Logf("Set repo OIDC claims with nativeID %s", nativeID)

	defer func() {
		_, _ = prov.Delete(ctx, &resource.DeleteRequest{
			NativeID:     nativeID,
			ResourceType: OIDCClaimsResourceType,
			TargetConfig: targetConfig,
		})
	}()

	// Read
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: OIDCClaimsResourceType,
		TargetConfig: targetConfig,
	})
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if readResult.Properties == "" {
		t.Fatal("Read returned empty properties")
	}
	var readProps repoOIDCClaimsProperties
	json.Unmarshal([]byte(readResult.Properties), &readProps)
	if len(readProps.IncludeClaimKeys) < 2 {
		t.Errorf("Read includeClaimKeys length = %d, want >= 2", len(readProps.IncludeClaimKeys))
	}

	// Update
	desiredProps, _ := json.Marshal(repoOIDCClaimsProperties{
		NativeID:         oidcClaimsNativeID,
		UseDefault:       github.Ptr(false),
		IncludeClaimKeys: []string{"repo", "context", "ref"},
	})
	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:          nativeID,
		ResourceType:      OIDCClaimsResourceType,
		DesiredProperties: desiredProps,
		TargetConfig:      targetConfig,
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}
	if updateResult.ProgressResult.OperationStatus != resource.OperationStatusSuccess {
		t.Fatalf("Update failed: %s", updateResult.ProgressResult.StatusMessage)
	}

	// List (singleton always returns the fixed ID)
	listResult, _ := prov.List(ctx, &resource.ListRequest{
		ResourceType: OIDCClaimsResourceType,
		TargetConfig: targetConfig,
	})
	if len(listResult.NativeIDs) != 1 || listResult.NativeIDs[0] != oidcClaimsNativeID {
		t.Errorf("List = %v, want [%s]", listResult.NativeIDs, oidcClaimsNativeID)
	}

	// Delete (reset to default)
	deleteResult, _ := prov.Delete(ctx, &resource.DeleteRequest{
		NativeID:     nativeID,
		ResourceType: OIDCClaimsResourceType,
		TargetConfig: targetConfig,
	})
	if deleteResult.ProgressResult.OperationStatus != resource.OperationStatusSuccess {
		t.Fatalf("Delete failed: %s", deleteResult.ProgressResult.StatusMessage)
	}

	// Read after reset — should still return properties (template reverts to default)
	readResult2, _ := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: OIDCClaimsResourceType,
		TargetConfig: targetConfig,
	})
	if readResult2.Properties == "" {
		t.Log("Read after delete returned empty (expected for default template)")
	} else {
		var readProps2 repoOIDCClaimsProperties
		json.Unmarshal([]byte(readResult2.Properties), &readProps2)
		if readProps2.UseDefault != nil && *readProps2.UseDefault {
			t.Log("Template reset to default successfully")
		}
	}
}
