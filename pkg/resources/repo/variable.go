// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: Apache-2.0

package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/go-github/v69/github"

	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/config"
	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/provisioner"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

const VariableResourceType = "GHA::Repo::Variable"

func init() {
	provisioner.Register(VariableResourceType, func(client *github.Client, cfg *config.Config) provisioner.Provisioner {
		return &variableProvisioner{client: client, cfg: cfg}
	})
}

type variableProperties struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type variableProvisioner struct {
	client *github.Client
	cfg    *config.Config
}

func (p *variableProvisioner) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	var props variableProperties
	if err := json.Unmarshal(req.Properties, &props); err != nil {
		return provisioner.CreateFailure(resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	normalizedName := strings.ToUpper(props.Name)

	_, err := p.client.Actions.CreateRepoVariable(ctx, p.cfg.Owner, p.cfg.Repo, &github.ActionsVariable{
		Name:  normalizedName,
		Value: props.Value,
	})
	if err != nil {
		return provisioner.CreateFailure(provisioner.ClassifyError(err), err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(variableProperties{Name: normalizedName, Value: props.Value})
	return provisioner.CreateSuccess(normalizedName, resultProps), nil
}

func (p *variableProvisioner) Read(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
	v, _, err := p.client.Actions.GetRepoVariable(ctx, p.cfg.Owner, p.cfg.Repo, req.NativeID)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.ReadNotFound(VariableResourceType), nil
		}
		return &resource.ReadResult{ResourceType: VariableResourceType, ErrorCode: provisioner.ClassifyError(err)}, nil
	}

	props := provisioner.MustMarshal(variableProperties{Name: v.Name, Value: v.Value})
	return provisioner.ReadSuccess(VariableResourceType, props), nil
}

func (p *variableProvisioner) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var desired variableProperties
	if err := json.Unmarshal(req.DesiredProperties, &desired); err != nil {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	_, err := p.client.Actions.UpdateRepoVariable(ctx, p.cfg.Owner, p.cfg.Repo, &github.ActionsVariable{
		Name:  req.NativeID,
		Value: desired.Value,
	})
	if err != nil {
		return provisioner.UpdateFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(variableProperties{Name: req.NativeID, Value: desired.Value})
	return provisioner.UpdateSuccess(req.NativeID, resultProps), nil
}

func (p *variableProvisioner) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	_, err := p.client.Actions.DeleteRepoVariable(ctx, p.cfg.Owner, p.cfg.Repo, req.NativeID)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.DeleteSuccess(req.NativeID), nil
		}
		return provisioner.DeleteFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	return provisioner.DeleteSuccess(req.NativeID), nil
}

func (p *variableProvisioner) List(ctx context.Context, req *resource.ListRequest) (*resource.ListResult, error) {
	var allIDs []string
	opts := &github.ListOptions{PerPage: 100}
	for {
		vars, resp, err := p.client.Actions.ListRepoVariables(ctx, p.cfg.Owner, p.cfg.Repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list repo variables: %w", err)
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

func (p *variableProvisioner) Status(_ context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	return provisioner.StatusSuccess(req.NativeID), nil
}
