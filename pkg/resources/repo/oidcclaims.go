// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: Apache-2.0

package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/go-github/v69/github"

	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/config"
	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/provisioner"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

const OIDCClaimsResourceType = "GHA::Repo::OIDCClaims"

const oidcClaimsNativeID = "oidc-claims"

func init() {
	provisioner.Register(OIDCClaimsResourceType, func(client *github.Client, cfg *config.Config) provisioner.Provisioner {
		return &repoOIDCClaimsProvisioner{client: client, cfg: cfg}
	})
}

type repoOIDCClaimsProperties struct {
	NativeID         string   `json:"nativeId"`
	UseDefault       *bool    `json:"useDefault,omitempty"`
	IncludeClaimKeys []string `json:"includeClaimKeys,omitempty"`
}

type repoOIDCClaimsProvisioner struct {
	client *github.Client
	cfg    *config.Config
}

func (p *repoOIDCClaimsProvisioner) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	var props repoOIDCClaimsProperties
	if err := json.Unmarshal(req.Properties, &props); err != nil {
		return provisioner.CreateFailure(resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	template := &github.OIDCSubjectClaimCustomTemplate{
		UseDefault:       props.UseDefault,
		IncludeClaimKeys: props.IncludeClaimKeys,
	}

	_, err := p.client.Actions.SetRepoOIDCSubjectClaimCustomTemplate(ctx, p.cfg.Owner, p.cfg.Repo, template)
	if err != nil {
		return provisioner.CreateFailure(provisioner.ClassifyError(err), err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(repoOIDCClaimsProperties{
		NativeID:         oidcClaimsNativeID,
		UseDefault:       props.UseDefault,
		IncludeClaimKeys: props.IncludeClaimKeys,
	})
	return provisioner.CreateSuccess(oidcClaimsNativeID, resultProps), nil
}

func (p *repoOIDCClaimsProvisioner) Read(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
	template, _, err := p.client.Actions.GetRepoOIDCSubjectClaimCustomTemplate(ctx, p.cfg.Owner, p.cfg.Repo)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.ReadNotFound(OIDCClaimsResourceType), nil
		}
		return &resource.ReadResult{ResourceType: OIDCClaimsResourceType, ErrorCode: provisioner.ClassifyError(err)}, nil
	}

	props := provisioner.MustMarshal(repoOIDCClaimsProperties{
		NativeID:         oidcClaimsNativeID,
		UseDefault:       template.UseDefault,
		IncludeClaimKeys: template.IncludeClaimKeys,
	})
	return provisioner.ReadSuccess(OIDCClaimsResourceType, props), nil
}

func (p *repoOIDCClaimsProvisioner) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var desired repoOIDCClaimsProperties
	if err := json.Unmarshal(req.DesiredProperties, &desired); err != nil {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInvalidRequest, err.Error()), nil
	}

	template := &github.OIDCSubjectClaimCustomTemplate{
		UseDefault:       desired.UseDefault,
		IncludeClaimKeys: desired.IncludeClaimKeys,
	}

	_, err := p.client.Actions.SetRepoOIDCSubjectClaimCustomTemplate(ctx, p.cfg.Owner, p.cfg.Repo, template)
	if err != nil {
		return provisioner.UpdateFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(repoOIDCClaimsProperties{
		NativeID:         oidcClaimsNativeID,
		UseDefault:       desired.UseDefault,
		IncludeClaimKeys: desired.IncludeClaimKeys,
	})
	return provisioner.UpdateSuccess(req.NativeID, resultProps), nil
}

func (p *repoOIDCClaimsProvisioner) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	template := &github.OIDCSubjectClaimCustomTemplate{
		UseDefault: github.Ptr(true),
	}

	_, err := p.client.Actions.SetRepoOIDCSubjectClaimCustomTemplate(ctx, p.cfg.Owner, p.cfg.Repo, template)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.DeleteSuccess(req.NativeID), nil
		}
		return provisioner.DeleteFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}
	return provisioner.DeleteSuccess(req.NativeID), nil
}

func (p *repoOIDCClaimsProvisioner) List(_ context.Context, _ *resource.ListRequest) (*resource.ListResult, error) {
	return &resource.ListResult{NativeIDs: []string{oidcClaimsNativeID}}, nil
}

func (p *repoOIDCClaimsProvisioner) Status(_ context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	return provisioner.StatusSuccess(req.NativeID), nil
}
