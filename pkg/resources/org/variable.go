// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

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

const VariableResourceType = "GHA::Org::Variable"

func init() {
	provisioner.Register(VariableResourceType, func(client *github.Client, cfg *config.Config) provisioner.Provisioner {
		return &orgVariableProvisioner{client: client, cfg: cfg}
	})
}

type orgVariableProperties struct {
	Name                  string  `json:"name"`
	Value                 string  `json:"value"`
	Visibility            string  `json:"visibility"`
	SelectedRepositoryIds []int64 `json:"selectedRepositoryIds,omitempty"`
}

type orgVariableProvisioner struct {
	client *github.Client
	cfg    *config.Config
}

func (p *orgVariableProvisioner) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	var props orgVariableProperties
	if err := json.Unmarshal(req.Properties, &props); err != nil {
		return provisioner.CreateFailure(resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	v := &github.ActionsVariable{
		Name:       props.Name,
		Value:      props.Value,
		Visibility: github.Ptr(props.Visibility),
	}
	if len(props.SelectedRepositoryIds) > 0 {
		ids := github.SelectedRepoIDs(props.SelectedRepositoryIds)
		v.SelectedRepositoryIDs = &ids
	}

	_, err := p.client.Actions.CreateOrgVariable(ctx, p.cfg.Owner, v)
	if err != nil {
		return provisioner.CreateFailure(provisioner.ClassifyError(err), err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(props)
	return provisioner.CreateSuccess(props.Name, resultProps), nil
}

func (p *orgVariableProvisioner) Read(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
	v, _, err := p.client.Actions.GetOrgVariable(ctx, p.cfg.Owner, req.NativeID)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.ReadNotFound(VariableResourceType), nil
		}
		return &resource.ReadResult{ResourceType: VariableResourceType, ErrorCode: provisioner.ClassifyError(err)}, nil
	}

	props := provisioner.MustMarshal(orgVariableProperties{
		Name:       v.Name,
		Value:      v.Value,
		Visibility: v.GetVisibility(),
	})
	return provisioner.ReadSuccess(VariableResourceType, props), nil
}

func (p *orgVariableProvisioner) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var desired orgVariableProperties
	if err := json.Unmarshal(req.DesiredProperties, &desired); err != nil {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInvalidRequest, err.Error()), nil
	}

	v := &github.ActionsVariable{
		Name:       req.NativeID,
		Value:      desired.Value,
		Visibility: github.Ptr(desired.Visibility),
	}
	if len(desired.SelectedRepositoryIds) > 0 {
		ids := github.SelectedRepoIDs(desired.SelectedRepositoryIds)
		v.SelectedRepositoryIDs = &ids
	}

	_, err := p.client.Actions.UpdateOrgVariable(ctx, p.cfg.Owner, v)
	if err != nil {
		return provisioner.UpdateFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(orgVariableProperties{
		Name:                  req.NativeID,
		Value:                 desired.Value,
		Visibility:            desired.Visibility,
		SelectedRepositoryIds: desired.SelectedRepositoryIds,
	})
	return provisioner.UpdateSuccess(req.NativeID, resultProps), nil
}

func (p *orgVariableProvisioner) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	_, err := p.client.Actions.DeleteOrgVariable(ctx, p.cfg.Owner, req.NativeID)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.DeleteSuccess(req.NativeID), nil
		}
		return provisioner.DeleteFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}
	return provisioner.DeleteSuccess(req.NativeID), nil
}

func (p *orgVariableProvisioner) List(ctx context.Context, _ *resource.ListRequest) (*resource.ListResult, error) {
	var allIDs []string
	opts := &github.ListOptions{PerPage: 100}
	for {
		vars, resp, err := p.client.Actions.ListOrgVariables(ctx, p.cfg.Owner, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list org variables: %w", err)
		}
		for _, v := range vars.Variables {
			allIDs = append(allIDs, v.Name)
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return &resource.ListResult{NativeIDs: allIDs}, nil
}

func (p *orgVariableProvisioner) Status(_ context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	return provisioner.StatusSuccess(req.NativeID), nil
}
