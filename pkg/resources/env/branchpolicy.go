// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package env

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v69/github"

	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/config"
	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/provisioner"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

const BranchPolicyResourceType = "GHA::Environment::BranchPolicy"

func init() {
	provisioner.Register(BranchPolicyResourceType, func(client *github.Client, cfg *config.Config) provisioner.Provisioner {
		return &branchPolicyProvisioner{client: client, cfg: cfg}
	})
}

type branchPolicyProps struct {
	Environment string `json:"environment"`
	Name        string `json:"name"`
	PolicyType  string `json:"policyType,omitempty"`
	ID          int64  `json:"id,omitempty"`
}

type branchPolicyProvisioner struct {
	client *github.Client
	cfg    *config.Config
}

func (p *branchPolicyProvisioner) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	var props branchPolicyProps
	if err := json.Unmarshal(req.Properties, &props); err != nil {
		return provisioner.CreateFailure(resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	policyType := props.PolicyType
	if policyType == "" {
		policyType = "branch"
	}

	policy, _, err := p.client.Repositories.CreateDeploymentBranchPolicy(
		ctx, p.cfg.Owner, p.cfg.Repo, props.Environment,
		&github.DeploymentBranchPolicyRequest{
			Name: github.Ptr(props.Name),
			Type: github.Ptr(policyType),
		},
	)
	if err != nil {
		return provisioner.CreateFailure(provisioner.ClassifyError(err), err.Error()), nil
	}

	nativeID := props.Environment + "/" + strconv.FormatInt(policy.GetID(), 10)
	resultProps := provisioner.MustMarshal(branchPolicyProps{
		Environment: props.Environment,
		Name:        policy.GetName(),
		PolicyType:  policy.GetType(),
		ID:          policy.GetID(),
	})
	return provisioner.CreateSuccess(nativeID, resultProps), nil
}

func (p *branchPolicyProvisioner) Read(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
	parts := strings.SplitN(req.NativeID, "/", 2)
	if len(parts) != 2 {
		return &resource.ReadResult{ResourceType: BranchPolicyResourceType, ErrorCode: resource.OperationErrorCodeInternalFailure}, nil
	}
	envName, idStr := parts[0], parts[1]
	policyID, _ := strconv.ParseInt(idStr, 10, 64)

	policy, _, err := p.client.Repositories.GetDeploymentBranchPolicy(ctx, p.cfg.Owner, p.cfg.Repo, envName, policyID)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.ReadNotFound(BranchPolicyResourceType), nil
		}
		return &resource.ReadResult{ResourceType: BranchPolicyResourceType, ErrorCode: provisioner.ClassifyError(err)}, nil
	}

	props := provisioner.MustMarshal(branchPolicyProps{
		Environment: envName,
		Name:        policy.GetName(),
		PolicyType:  policy.GetType(),
		ID:          policy.GetID(),
	})
	return provisioner.ReadSuccess(BranchPolicyResourceType, props), nil
}

func (p *branchPolicyProvisioner) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	parts := strings.SplitN(req.NativeID, "/", 2)
	if len(parts) != 2 {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInternalFailure, fmt.Sprintf("invalid native ID %q", req.NativeID)), nil
	}
	envName, idStr := parts[0], parts[1]
	policyID, _ := strconv.ParseInt(idStr, 10, 64)

	var desired branchPolicyProps
	if err := json.Unmarshal(req.DesiredProperties, &desired); err != nil {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInvalidRequest, err.Error()), nil
	}

	policy, _, err := p.client.Repositories.UpdateDeploymentBranchPolicy(
		ctx, p.cfg.Owner, p.cfg.Repo, envName, policyID,
		&github.DeploymentBranchPolicyRequest{
			Name: github.Ptr(desired.Name),
		},
	)
	if err != nil {
		return provisioner.UpdateFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(branchPolicyProps{
		Environment: envName,
		Name:        policy.GetName(),
		PolicyType:  policy.GetType(),
		ID:          policy.GetID(),
	})
	return provisioner.UpdateSuccess(req.NativeID, resultProps), nil
}

func (p *branchPolicyProvisioner) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	parts := strings.SplitN(req.NativeID, "/", 2)
	if len(parts) != 2 {
		return provisioner.DeleteFailure(req.NativeID, resource.OperationErrorCodeInternalFailure, fmt.Sprintf("invalid native ID %q", req.NativeID)), nil
	}
	envName, idStr := parts[0], parts[1]
	policyID, _ := strconv.ParseInt(idStr, 10, 64)

	_, err := p.client.Repositories.DeleteDeploymentBranchPolicy(ctx, p.cfg.Owner, p.cfg.Repo, envName, policyID)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.DeleteSuccess(req.NativeID), nil
		}
		return provisioner.DeleteFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}
	return provisioner.DeleteSuccess(req.NativeID), nil
}

func (p *branchPolicyProvisioner) List(ctx context.Context, req *resource.ListRequest) (*resource.ListResult, error) {
	envName, ok := req.AdditionalProperties["environment"]
	if !ok || envName == "" {
		return nil, fmt.Errorf("environment is required to list branch policies")
	}

	resp, _, err := p.client.Repositories.ListDeploymentBranchPolicies(ctx, p.cfg.Owner, p.cfg.Repo, envName)
	if err != nil {
		return nil, err
	}

	allIDs := make([]string, 0, len(resp.BranchPolicies))
	for _, policy := range resp.BranchPolicies {
		allIDs = append(allIDs, envName+"/"+strconv.FormatInt(policy.GetID(), 10))
	}
	return &resource.ListResult{NativeIDs: allIDs}, nil
}

func (p *branchPolicyProvisioner) Status(_ context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	return provisioner.StatusSuccess(req.NativeID), nil
}
