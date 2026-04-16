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

const OrgOIDCClaimsResourceType = "GHA::Org::OIDCClaims"

const orgOIDCClaimsNativeID = "oidc-claims"

func init() {
	provisioner.Register(OrgOIDCClaimsResourceType, func(client *github.Client, cfg *config.Config) provisioner.Provisioner {
		return &orgOIDCClaimsProvisioner{client: client, cfg: cfg}
	})
}

type orgOIDCClaimsProperties struct {
	NativeID         string   `json:"nativeId"`
	IncludeClaimKeys []string `json:"includeClaimKeys,omitempty"`
}

type orgOIDCClaimsProvisioner struct {
	client *github.Client
	cfg    *config.Config
}

func (p *orgOIDCClaimsProvisioner) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	var props orgOIDCClaimsProperties
	if err := json.Unmarshal(req.Properties, &props); err != nil {
		return provisioner.CreateFailure(resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	template := &github.OIDCSubjectClaimCustomTemplate{
		IncludeClaimKeys: props.IncludeClaimKeys,
	}

	_, err := p.client.Actions.SetOrgOIDCSubjectClaimCustomTemplate(ctx, p.cfg.Owner, template)
	if err != nil {
		return provisioner.CreateFailure(provisioner.ClassifyError(err), err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(orgOIDCClaimsProperties{
		NativeID:         orgOIDCClaimsNativeID,
		IncludeClaimKeys: props.IncludeClaimKeys,
	})
	return provisioner.CreateSuccess(orgOIDCClaimsNativeID, resultProps), nil
}

func (p *orgOIDCClaimsProvisioner) Read(ctx context.Context, _ *resource.ReadRequest) (*resource.ReadResult, error) {
	template, _, err := p.client.Actions.GetOrgOIDCSubjectClaimCustomTemplate(ctx, p.cfg.Owner)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.ReadNotFound(OrgOIDCClaimsResourceType), nil
		}
		return &resource.ReadResult{ResourceType: OrgOIDCClaimsResourceType, ErrorCode: provisioner.ClassifyError(err)}, nil
	}

	props := provisioner.MustMarshal(orgOIDCClaimsProperties{
		NativeID:         orgOIDCClaimsNativeID,
		IncludeClaimKeys: template.IncludeClaimKeys,
	})
	return provisioner.ReadSuccess(OrgOIDCClaimsResourceType, props), nil
}

func (p *orgOIDCClaimsProvisioner) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var desired orgOIDCClaimsProperties
	if err := json.Unmarshal(req.DesiredProperties, &desired); err != nil {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInvalidRequest, err.Error()), nil
	}

	template := &github.OIDCSubjectClaimCustomTemplate{
		IncludeClaimKeys: desired.IncludeClaimKeys,
	}

	_, err := p.client.Actions.SetOrgOIDCSubjectClaimCustomTemplate(ctx, p.cfg.Owner, template)
	if err != nil {
		return provisioner.UpdateFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(orgOIDCClaimsProperties{
		NativeID:         orgOIDCClaimsNativeID,
		IncludeClaimKeys: desired.IncludeClaimKeys,
	})
	return provisioner.UpdateSuccess(req.NativeID, resultProps), nil
}

func (p *orgOIDCClaimsProvisioner) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	template := &github.OIDCSubjectClaimCustomTemplate{
		IncludeClaimKeys: []string{},
	}

	_, err := p.client.Actions.SetOrgOIDCSubjectClaimCustomTemplate(ctx, p.cfg.Owner, template)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.DeleteSuccess(req.NativeID), nil
		}
		return provisioner.DeleteFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}
	return provisioner.DeleteSuccess(req.NativeID), nil
}

func (p *orgOIDCClaimsProvisioner) List(_ context.Context, _ *resource.ListRequest) (*resource.ListResult, error) {
	return &resource.ListResult{NativeIDs: []string{orgOIDCClaimsNativeID}}, nil
}

func (p *orgOIDCClaimsProvisioner) Status(_ context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	return provisioner.StatusSuccess(req.NativeID), nil
}
