// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package env

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

const VariableResourceType = "GHA::Environment::Variable"

func init() {
	provisioner.Register(VariableResourceType, func(client *github.Client, cfg *config.Config) provisioner.Provisioner {
		return &envVariableProvisioner{client: client, cfg: cfg}
	})
}

type envVariableProperties struct {
	Environment string `json:"environment"`
	Name        string `json:"name"`
	Value       string `json:"value"`
}

type envVariableProvisioner struct {
	client *github.Client
	cfg    *config.Config
}

func (p *envVariableProvisioner) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	var props envVariableProperties
	if err := json.Unmarshal(req.Properties, &props); err != nil {
		return provisioner.CreateFailure(resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	_, err := p.client.Actions.CreateEnvVariable(ctx, p.cfg.Owner, p.cfg.Repo, props.Environment, &github.ActionsVariable{
		Name:  props.Name,
		Value: props.Value,
	})
	if err != nil {
		return provisioner.CreateFailure(provisioner.ClassifyError(err), err.Error()), nil
	}

	nativeID := props.Environment + "/" + props.Name
	resultProps := provisioner.MustMarshal(props)
	return provisioner.CreateSuccess(nativeID, resultProps), nil
}

func (p *envVariableProvisioner) Read(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
	parts := strings.SplitN(req.NativeID, "/", 2)
	if len(parts) != 2 {
		return &resource.ReadResult{ResourceType: VariableResourceType, ErrorCode: resource.OperationErrorCodeInternalFailure}, nil
	}
	envName, name := parts[0], parts[1]

	v, _, err := p.client.Actions.GetEnvVariable(ctx, p.cfg.Owner, p.cfg.Repo, envName, name)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.ReadNotFound(VariableResourceType), nil
		}
		return &resource.ReadResult{ResourceType: VariableResourceType, ErrorCode: provisioner.ClassifyError(err)}, nil
	}

	props := provisioner.MustMarshal(envVariableProperties{
		Environment: envName,
		Name:        v.Name,
		Value:       v.Value,
	})
	return provisioner.ReadSuccess(VariableResourceType, props), nil
}

func (p *envVariableProvisioner) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	parts := strings.SplitN(req.NativeID, "/", 2)
	if len(parts) != 2 {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInternalFailure, fmt.Sprintf("invalid native ID %q", req.NativeID)), nil
	}
	envName, name := parts[0], parts[1]

	var desired envVariableProperties
	if err := json.Unmarshal(req.DesiredProperties, &desired); err != nil {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInvalidRequest, err.Error()), nil
	}

	_, err := p.client.Actions.UpdateEnvVariable(ctx, p.cfg.Owner, p.cfg.Repo, envName, &github.ActionsVariable{
		Name:  name,
		Value: desired.Value,
	})
	if err != nil {
		return provisioner.UpdateFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(envVariableProperties{Environment: envName, Name: name, Value: desired.Value})
	return provisioner.UpdateSuccess(req.NativeID, resultProps), nil
}

func (p *envVariableProvisioner) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	parts := strings.SplitN(req.NativeID, "/", 2)
	if len(parts) != 2 {
		return provisioner.DeleteFailure(req.NativeID, resource.OperationErrorCodeInternalFailure, fmt.Sprintf("invalid native ID %q", req.NativeID)), nil
	}
	envName, name := parts[0], parts[1]

	_, err := p.client.Actions.DeleteEnvVariable(ctx, p.cfg.Owner, p.cfg.Repo, envName, name)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.DeleteSuccess(req.NativeID), nil
		}
		return provisioner.DeleteFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}
	return provisioner.DeleteSuccess(req.NativeID), nil
}

func (p *envVariableProvisioner) List(ctx context.Context, req *resource.ListRequest) (*resource.ListResult, error) {
	envName, ok := req.AdditionalProperties["environment"]
	if !ok || envName == "" {
		return nil, fmt.Errorf("environment is required to list environment variables")
	}

	var allIDs []string
	opts := &github.ListOptions{PerPage: 100}
	for {
		vars, resp, err := p.client.Actions.ListEnvVariables(ctx, p.cfg.Owner, p.cfg.Repo, envName, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list env variables: %w", err)
		}
		for _, v := range vars.Variables {
			allIDs = append(allIDs, envName+"/"+v.Name)
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return &resource.ListResult{NativeIDs: allIDs}, nil
}

func (p *envVariableProvisioner) Status(_ context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	return provisioner.StatusSuccess(req.NativeID), nil
}
