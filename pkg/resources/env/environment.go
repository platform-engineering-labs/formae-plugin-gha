// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package env

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/go-github/v69/github"

	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/config"
	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/provisioner"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)


const EnvironmentResourceType = "GHA::Repo::Environment"

func init() {
	provisioner.Register(EnvironmentResourceType, func(client *github.Client, cfg *config.Config) provisioner.Provisioner {
		return &environmentProvisioner{client: client, cfg: cfg}
	})
}

// environmentProperties is the JSON structure for an environment.
type environmentProperties struct {
	Name                   string                    `json:"name"`
	WaitTimer              *int                      `json:"waitTimer,omitempty"`
	PreventSelfReview      *bool                     `json:"preventSelfReview,omitempty"`
	CanAdminsBypass        *bool                     `json:"canAdminsBypass,omitempty"`
	Reviewers              []reviewerProperties      `json:"reviewers,omitempty"`
	DeploymentBranchPolicy *branchPolicyProperties   `json:"deploymentBranchPolicy,omitempty"`
}

type reviewerProperties struct {
	Type string `json:"type"`
	ID   int64  `json:"id"`
}

type branchPolicyProperties struct {
	ProtectedBranches    bool `json:"protectedBranches"`
	CustomBranchPolicies bool `json:"customBranchPolicies"`
}

type environmentProvisioner struct {
	client *github.Client
	cfg    *config.Config
}

func (p *environmentProvisioner) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	var props environmentProperties
	if err := json.Unmarshal(req.Properties, &props); err != nil {
		return provisioner.CreateFailure(resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	input := toCreateUpdateEnvironment(&props)
	env, _, err := p.client.Repositories.CreateUpdateEnvironment(ctx, p.cfg.Owner, p.cfg.Repo, props.Name, input)
	if err != nil {
		code := provisioner.ClassifyError(err)
		return provisioner.CreateFailure(code, err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(environmentFromGitHub(env))
	return provisioner.CreateSuccess(props.Name, resultProps), nil
}

func (p *environmentProvisioner) Read(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
	env, _, err := p.client.Repositories.GetEnvironment(ctx, p.cfg.Owner, p.cfg.Repo, req.NativeID)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.ReadNotFound(EnvironmentResourceType), nil
		}
		return &resource.ReadResult{
			ResourceType: EnvironmentResourceType,
			ErrorCode:    provisioner.ClassifyError(err),
		}, nil
	}

	props := provisioner.MustMarshal(environmentFromGitHub(env))
	return provisioner.ReadSuccess(EnvironmentResourceType, props), nil
}

func (p *environmentProvisioner) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var desired environmentProperties
	if err := json.Unmarshal(req.DesiredProperties, &desired); err != nil {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	input := toCreateUpdateEnvironment(&desired)
	env, _, err := p.client.Repositories.CreateUpdateEnvironment(ctx, p.cfg.Owner, p.cfg.Repo, req.NativeID, input)
	if err != nil {
		code := provisioner.ClassifyError(err)
		return provisioner.UpdateFailure(req.NativeID, code, err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(environmentFromGitHub(env))
	return provisioner.UpdateSuccess(req.NativeID, resultProps), nil
}

func (p *environmentProvisioner) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	_, err := p.client.Repositories.DeleteEnvironment(ctx, p.cfg.Owner, p.cfg.Repo, req.NativeID)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.DeleteSuccess(req.NativeID), nil
		}
		code := provisioner.ClassifyError(err)
		return provisioner.DeleteFailure(req.NativeID, code, err.Error()), nil
	}

	return provisioner.DeleteSuccess(req.NativeID), nil
}

func (p *environmentProvisioner) List(ctx context.Context, req *resource.ListRequest) (*resource.ListResult, error) {
	var allIDs []string
	opts := &github.EnvironmentListOptions{ListOptions: github.ListOptions{PerPage: 100}}

	for {
		envs, resp, err := p.client.Repositories.ListEnvironments(ctx, p.cfg.Owner, p.cfg.Repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list environments: %w", err)
		}
		for _, e := range envs.Environments {
			allIDs = append(allIDs, e.GetName())
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return &resource.ListResult{NativeIDs: allIDs}, nil
}

func (p *environmentProvisioner) Status(_ context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	return provisioner.StatusSuccess(req.NativeID), nil
}

// toCreateUpdateEnvironment converts our properties to the go-github input type.
func toCreateUpdateEnvironment(props *environmentProperties) *github.CreateUpdateEnvironment {
	input := &github.CreateUpdateEnvironment{
		WaitTimer:         props.WaitTimer,
		PreventSelfReview: props.PreventSelfReview,
		CanAdminsBypass:   props.CanAdminsBypass,
	}

	if len(props.Reviewers) > 0 {
		reviewers := make([]*github.EnvReviewers, len(props.Reviewers))
		for i, r := range props.Reviewers {
			reviewers[i] = &github.EnvReviewers{
				Type: github.Ptr(r.Type),
				ID:   github.Ptr(r.ID),
			}
		}
		input.Reviewers = reviewers
	}

	if props.DeploymentBranchPolicy != nil {
		input.DeploymentBranchPolicy = &github.BranchPolicy{
			ProtectedBranches:    github.Ptr(props.DeploymentBranchPolicy.ProtectedBranches),
			CustomBranchPolicies: github.Ptr(props.DeploymentBranchPolicy.CustomBranchPolicies),
		}
	}

	return input
}

// environmentFromGitHub normalizes a GitHub environment response to our properties format.
// GitHub returns expanded reviewer objects; we flatten to {type, id}.
func environmentFromGitHub(env *github.Environment) environmentProperties {
	props := environmentProperties{
		Name:            env.GetName(),
		CanAdminsBypass: env.CanAdminsBypass,
	}

	// Extract wait_timer and reviewers from protection rules
	for _, rule := range env.ProtectionRules {
		switch rule.GetType() {
		case "wait_timer":
			props.WaitTimer = rule.WaitTimer
		case "required_reviewers":
			props.PreventSelfReview = rule.PreventSelfReview
			for _, r := range rule.Reviewers {
				var id int64
				switch reviewer := r.Reviewer.(type) {
				case *github.User:
					id = reviewer.GetID()
				case *github.Team:
					id = reviewer.GetID()
				}
				if id > 0 {
					props.Reviewers = append(props.Reviewers, reviewerProperties{
						Type: r.GetType(),
						ID:   id,
					})
				}
			}
		}
	}

	if env.DeploymentBranchPolicy != nil {
		props.DeploymentBranchPolicy = &branchPolicyProperties{
			ProtectedBranches:    env.DeploymentBranchPolicy.GetProtectedBranches(),
			CustomBranchPolicies: env.DeploymentBranchPolicy.GetCustomBranchPolicies(),
		}
	}

	return props
}
