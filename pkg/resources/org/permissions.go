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

const PermissionsResourceType = "GHA::Org::Permissions"

const permissionsNativeID = "permissions"

func init() {
	provisioner.Register(PermissionsResourceType, func(client *github.Client, cfg *config.Config) provisioner.Provisioner {
		return &permissionsProvisioner{client: client, cfg: cfg}
	})
}

type permissionsProperties struct {
	NativeID            string `json:"nativeId"`
	EnabledRepositories string `json:"enabledRepositories,omitempty"`
	AllowedActions      string `json:"allowedActions,omitempty"`
}

type permissionsProvisioner struct {
	client *github.Client
	cfg    *config.Config
}

func (p *permissionsProvisioner) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	var props permissionsProperties
	if err := json.Unmarshal(req.Properties, &props); err != nil {
		return provisioner.CreateFailure(resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	perms := github.ActionsPermissions{
		EnabledRepositories: github.Ptr(props.EnabledRepositories),
		AllowedActions:      github.Ptr(props.AllowedActions),
	}

	result, _, err := p.client.Actions.EditActionsPermissions(ctx, p.cfg.Owner, perms)
	if err != nil {
		return provisioner.CreateFailure(provisioner.ClassifyError(err), err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(permissionsProperties{
		NativeID:            permissionsNativeID,
		EnabledRepositories: result.GetEnabledRepositories(),
		AllowedActions:      result.GetAllowedActions(),
	})
	return provisioner.CreateSuccess(permissionsNativeID, resultProps), nil
}

func (p *permissionsProvisioner) Read(ctx context.Context, _ *resource.ReadRequest) (*resource.ReadResult, error) {
	perms, _, err := p.client.Actions.GetActionsPermissions(ctx, p.cfg.Owner)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.ReadNotFound(PermissionsResourceType), nil
		}
		return &resource.ReadResult{ResourceType: PermissionsResourceType, ErrorCode: provisioner.ClassifyError(err)}, nil
	}

	props := provisioner.MustMarshal(permissionsProperties{
		NativeID:            permissionsNativeID,
		EnabledRepositories: perms.GetEnabledRepositories(),
		AllowedActions:      perms.GetAllowedActions(),
	})
	return provisioner.ReadSuccess(PermissionsResourceType, props), nil
}

func (p *permissionsProvisioner) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var desired permissionsProperties
	if err := json.Unmarshal(req.DesiredProperties, &desired); err != nil {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInvalidRequest, err.Error()), nil
	}

	perms := github.ActionsPermissions{
		EnabledRepositories: github.Ptr(desired.EnabledRepositories),
		AllowedActions:      github.Ptr(desired.AllowedActions),
	}

	result, _, err := p.client.Actions.EditActionsPermissions(ctx, p.cfg.Owner, perms)
	if err != nil {
		return provisioner.UpdateFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(permissionsProperties{
		NativeID:            permissionsNativeID,
		EnabledRepositories: result.GetEnabledRepositories(),
		AllowedActions:      result.GetAllowedActions(),
	})
	return provisioner.UpdateSuccess(req.NativeID, resultProps), nil
}

func (p *permissionsProvisioner) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	perms := github.ActionsPermissions{
		EnabledRepositories: github.Ptr("all"),
		AllowedActions:      github.Ptr("all"),
	}

	_, _, err := p.client.Actions.EditActionsPermissions(ctx, p.cfg.Owner, perms)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.DeleteSuccess(req.NativeID), nil
		}
		return provisioner.DeleteFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}
	return provisioner.DeleteSuccess(req.NativeID), nil
}

func (p *permissionsProvisioner) List(_ context.Context, _ *resource.ListRequest) (*resource.ListResult, error) {
	return &resource.ListResult{NativeIDs: []string{permissionsNativeID}}, nil
}

func (p *permissionsProvisioner) Status(_ context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	return provisioner.StatusSuccess(req.NativeID), nil
}
