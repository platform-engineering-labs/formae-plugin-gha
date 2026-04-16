// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package org

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/google/go-github/v69/github"

	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/config"
	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/provisioner"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

const RunnerGroupResourceType = "GHA::Org::RunnerGroup"

func init() {
	provisioner.Register(RunnerGroupResourceType, func(client *github.Client, cfg *config.Config) provisioner.Provisioner {
		return &runnerGroupProvisioner{client: client, cfg: cfg}
	})
}

type runnerGroupProperties struct {
	Name                     string   `json:"name"`
	Visibility               string   `json:"visibility,omitempty"`
	SelectedRepositoryIds    []int64  `json:"selectedRepositoryIds,omitempty"`
	AllowsPublicRepositories *bool    `json:"allowsPublicRepositories,omitempty"`
	RestrictedToWorkflows    *bool    `json:"restrictedToWorkflows,omitempty"`
	SelectedWorkflows        []string `json:"selectedWorkflows,omitempty"`
	ID                       *int64   `json:"id,omitempty"`
}

type runnerGroupProvisioner struct {
	client *github.Client
	cfg    *config.Config
}

func runnerGroupFromGitHub(rg *github.RunnerGroup) runnerGroupProperties {
	props := runnerGroupProperties{
		Name:                     rg.GetName(),
		Visibility:               rg.GetVisibility(),
		AllowsPublicRepositories: rg.AllowsPublicRepositories,
		RestrictedToWorkflows:    rg.RestrictedToWorkflows,
		SelectedWorkflows:        rg.SelectedWorkflows,
		ID:                       rg.ID,
	}
	return props
}

func (p *runnerGroupProvisioner) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	var props runnerGroupProperties
	if err := json.Unmarshal(req.Properties, &props); err != nil {
		return provisioner.CreateFailure(resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	createReq := github.CreateRunnerGroupRequest{
		Name:                     github.Ptr(props.Name),
		Visibility:               github.Ptr(props.Visibility),
		AllowsPublicRepositories: props.AllowsPublicRepositories,
		RestrictedToWorkflows:    props.RestrictedToWorkflows,
		SelectedWorkflows:        props.SelectedWorkflows,
		SelectedRepositoryIDs:    props.SelectedRepositoryIds,
	}

	rg, _, err := p.client.Actions.CreateOrganizationRunnerGroup(ctx, p.cfg.Owner, createReq)
	if err != nil {
		return provisioner.CreateFailure(provisioner.ClassifyError(err), err.Error()), nil
	}

	nativeID := strconv.FormatInt(rg.GetID(), 10)
	resultProps := provisioner.MustMarshal(runnerGroupFromGitHub(rg))
	return provisioner.CreateSuccess(nativeID, resultProps), nil
}

func (p *runnerGroupProvisioner) Read(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
	groupID, err := strconv.ParseInt(req.NativeID, 10, 64)
	if err != nil {
		return &resource.ReadResult{ResourceType: RunnerGroupResourceType, ErrorCode: resource.OperationErrorCodeInvalidRequest}, nil
	}

	rg, _, err := p.client.Actions.GetOrganizationRunnerGroup(ctx, p.cfg.Owner, groupID)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.ReadNotFound(RunnerGroupResourceType), nil
		}
		return &resource.ReadResult{ResourceType: RunnerGroupResourceType, ErrorCode: provisioner.ClassifyError(err)}, nil
	}

	props := provisioner.MustMarshal(runnerGroupFromGitHub(rg))
	return provisioner.ReadSuccess(RunnerGroupResourceType, props), nil
}

func (p *runnerGroupProvisioner) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var desired runnerGroupProperties
	if err := json.Unmarshal(req.DesiredProperties, &desired); err != nil {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInvalidRequest, err.Error()), nil
	}

	groupID, err := strconv.ParseInt(req.NativeID, 10, 64)
	if err != nil {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInvalidRequest, err.Error()), nil
	}

	updateReq := github.UpdateRunnerGroupRequest{
		Name:                     github.Ptr(desired.Name),
		Visibility:               github.Ptr(desired.Visibility),
		AllowsPublicRepositories: desired.AllowsPublicRepositories,
		RestrictedToWorkflows:    desired.RestrictedToWorkflows,
		SelectedWorkflows:        desired.SelectedWorkflows,
	}

	rg, _, err := p.client.Actions.UpdateOrganizationRunnerGroup(ctx, p.cfg.Owner, groupID, updateReq)
	if err != nil {
		return provisioner.UpdateFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(runnerGroupFromGitHub(rg))
	return provisioner.UpdateSuccess(req.NativeID, resultProps), nil
}

func (p *runnerGroupProvisioner) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	groupID, err := strconv.ParseInt(req.NativeID, 10, 64)
	if err != nil {
		return provisioner.DeleteFailure(req.NativeID, resource.OperationErrorCodeInvalidRequest, err.Error()), nil
	}

	_, err = p.client.Actions.DeleteOrganizationRunnerGroup(ctx, p.cfg.Owner, groupID)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.DeleteSuccess(req.NativeID), nil
		}
		return provisioner.DeleteFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}
	return provisioner.DeleteSuccess(req.NativeID), nil
}

func (p *runnerGroupProvisioner) List(ctx context.Context, _ *resource.ListRequest) (*resource.ListResult, error) {
	var allIDs []string
	opts := &github.ListOrgRunnerGroupOptions{ListOptions: github.ListOptions{PerPage: 100}}
	for {
		groups, resp, err := p.client.Actions.ListOrganizationRunnerGroups(ctx, p.cfg.Owner, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list org runner groups: %w", err)
		}
		for _, rg := range groups.RunnerGroups {
			allIDs = append(allIDs, strconv.FormatInt(rg.GetID(), 10))
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return &resource.ListResult{NativeIDs: allIDs}, nil
}

func (p *runnerGroupProvisioner) Status(_ context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	return provisioner.StatusSuccess(req.NativeID), nil
}
