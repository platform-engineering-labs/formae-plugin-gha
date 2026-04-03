// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: Apache-2.0

package org

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/go-github/v69/github"

	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/config"
	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/provisioner"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

const WorkflowPermissionsResourceType = "GHA::Org::DefaultWorkflowPermissions"

const workflowPermissionsNativeID = "workflow-permissions"

func init() {
	provisioner.Register(WorkflowPermissionsResourceType, func(client *github.Client, cfg *config.Config) provisioner.Provisioner {
		return &workflowPermissionsProvisioner{client: client, cfg: cfg}
	})
}

type workflowPermissionsProperties struct {
	NativeID                     string `json:"nativeId"`
	DefaultWorkflowPermissions   string `json:"defaultWorkflowPermissions,omitempty"`
	CanApprovePullRequestReviews *bool  `json:"canApprovePullRequestReviews,omitempty"`
}

type workflowPermissionsProvisioner struct {
	client *github.Client
	cfg    *config.Config
}

func (p *workflowPermissionsProvisioner) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	var props workflowPermissionsProperties
	if err := json.Unmarshal(req.Properties, &props); err != nil {
		return provisioner.CreateFailure(resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	perms := github.DefaultWorkflowPermissionOrganization{
		DefaultWorkflowPermissions:   github.Ptr(props.DefaultWorkflowPermissions),
		CanApprovePullRequestReviews: props.CanApprovePullRequestReviews,
	}

	result, _, err := p.client.Actions.EditDefaultWorkflowPermissionsInOrganization(ctx, p.cfg.Owner, perms)
	if err != nil {
		return provisioner.CreateFailure(provisioner.ClassifyError(err), err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(workflowPermissionsProperties{
		NativeID:                     workflowPermissionsNativeID,
		DefaultWorkflowPermissions:   result.GetDefaultWorkflowPermissions(),
		CanApprovePullRequestReviews: result.CanApprovePullRequestReviews,
	})
	return provisioner.CreateSuccess(workflowPermissionsNativeID, resultProps), nil
}

func (p *workflowPermissionsProvisioner) Read(ctx context.Context, _ *resource.ReadRequest) (*resource.ReadResult, error) {
	perms, _, err := p.client.Actions.GetDefaultWorkflowPermissionsInOrganization(ctx, p.cfg.Owner)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.ReadNotFound(WorkflowPermissionsResourceType), nil
		}
		return &resource.ReadResult{ResourceType: WorkflowPermissionsResourceType, ErrorCode: provisioner.ClassifyError(err)}, nil
	}

	props := provisioner.MustMarshal(workflowPermissionsProperties{
		NativeID:                     workflowPermissionsNativeID,
		DefaultWorkflowPermissions:   perms.GetDefaultWorkflowPermissions(),
		CanApprovePullRequestReviews: perms.CanApprovePullRequestReviews,
	})
	return provisioner.ReadSuccess(WorkflowPermissionsResourceType, props), nil
}

func (p *workflowPermissionsProvisioner) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var desired workflowPermissionsProperties
	if err := json.Unmarshal(req.DesiredProperties, &desired); err != nil {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInvalidRequest, err.Error()), nil
	}

	perms := github.DefaultWorkflowPermissionOrganization{
		DefaultWorkflowPermissions:   github.Ptr(desired.DefaultWorkflowPermissions),
		CanApprovePullRequestReviews: desired.CanApprovePullRequestReviews,
	}

	result, _, err := p.client.Actions.EditDefaultWorkflowPermissionsInOrganization(ctx, p.cfg.Owner, perms)
	if err != nil {
		return provisioner.UpdateFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(workflowPermissionsProperties{
		NativeID:                     workflowPermissionsNativeID,
		DefaultWorkflowPermissions:   result.GetDefaultWorkflowPermissions(),
		CanApprovePullRequestReviews: result.CanApprovePullRequestReviews,
	})
	return provisioner.UpdateSuccess(req.NativeID, resultProps), nil
}

func (p *workflowPermissionsProvisioner) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	perms := github.DefaultWorkflowPermissionOrganization{
		DefaultWorkflowPermissions:   github.Ptr("read"),
		CanApprovePullRequestReviews: github.Ptr(false),
	}

	_, _, err := p.client.Actions.EditDefaultWorkflowPermissionsInOrganization(ctx, p.cfg.Owner, perms)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.DeleteSuccess(req.NativeID), nil
		}
		return provisioner.DeleteFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}
	return provisioner.DeleteSuccess(req.NativeID), nil
}

func (p *workflowPermissionsProvisioner) List(_ context.Context, _ *resource.ListRequest) (*resource.ListResult, error) {
	return &resource.ListResult{NativeIDs: []string{workflowPermissionsNativeID}}, nil
}

func (p *workflowPermissionsProvisioner) Status(_ context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	return provisioner.StatusSuccess(req.NativeID), nil
}
