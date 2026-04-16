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
	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/crypto"
	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/provisioner"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

const SecretResourceType = "GHA::Org::Secret"

func init() {
	provisioner.Register(SecretResourceType, func(client *github.Client, cfg *config.Config) provisioner.Provisioner {
		return &orgSecretProvisioner{client: client, cfg: cfg}
	})
}

type orgSecretProperties struct {
	Name                  string  `json:"name"`
	Value                 string  `json:"value,omitempty"`
	ValueHash             string  `json:"valueHash,omitempty"`
	Visibility            string  `json:"visibility"`
	SelectedRepositoryIds []int64 `json:"selectedRepositoryIds,omitempty"`
}

type orgSecretProvisioner struct {
	client *github.Client
	cfg    *config.Config
}

func (p *orgSecretProvisioner) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	var props orgSecretProperties
	if err := json.Unmarshal(req.Properties, &props); err != nil {
		return provisioner.CreateFailure(resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	if err := p.createOrUpdate(ctx, p.cfg.Owner, props.Name, props.Value, props.Visibility, props.SelectedRepositoryIds); err != nil {
		return provisioner.CreateFailure(provisioner.ClassifyError(err), err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(orgSecretProperties{
		Name:                  props.Name,
		ValueHash:             crypto.HashSecret(props.Value),
		Visibility:            props.Visibility,
		SelectedRepositoryIds: props.SelectedRepositoryIds,
	})
	return provisioner.CreateSuccess(props.Name, resultProps), nil
}

func (p *orgSecretProvisioner) Read(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
	s, _, err := p.client.Actions.GetOrgSecret(ctx, p.cfg.Owner, req.NativeID)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.ReadNotFound(SecretResourceType), nil
		}
		return &resource.ReadResult{ResourceType: SecretResourceType, ErrorCode: provisioner.ClassifyError(err)}, nil
	}

	props := provisioner.MustMarshal(orgSecretProperties{
		Name:       s.Name,
		Visibility: s.Visibility,
	})
	return provisioner.ReadSuccess(SecretResourceType, props), nil
}

func (p *orgSecretProvisioner) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var desired orgSecretProperties
	if err := json.Unmarshal(req.DesiredProperties, &desired); err != nil {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInvalidRequest, err.Error()), nil
	}

	if err := p.createOrUpdate(ctx, p.cfg.Owner, req.NativeID, desired.Value, desired.Visibility, desired.SelectedRepositoryIds); err != nil {
		return provisioner.UpdateFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(orgSecretProperties{
		Name:                  req.NativeID,
		ValueHash:             crypto.HashSecret(desired.Value),
		Visibility:            desired.Visibility,
		SelectedRepositoryIds: desired.SelectedRepositoryIds,
	})
	return provisioner.UpdateSuccess(req.NativeID, resultProps), nil
}

func (p *orgSecretProvisioner) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	_, err := p.client.Actions.DeleteOrgSecret(ctx, p.cfg.Owner, req.NativeID)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.DeleteSuccess(req.NativeID), nil
		}
		return provisioner.DeleteFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}
	return provisioner.DeleteSuccess(req.NativeID), nil
}

func (p *orgSecretProvisioner) List(ctx context.Context, _ *resource.ListRequest) (*resource.ListResult, error) {
	var allIDs []string
	opts := &github.ListOptions{PerPage: 100}
	for {
		secrets, resp, err := p.client.Actions.ListOrgSecrets(ctx, p.cfg.Owner, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list org secrets: %w", err)
		}
		for _, s := range secrets.Secrets {
			allIDs = append(allIDs, s.Name)
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return &resource.ListResult{NativeIDs: allIDs}, nil
}

func (p *orgSecretProvisioner) Status(_ context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	return provisioner.StatusSuccess(req.NativeID), nil
}

func (p *orgSecretProvisioner) createOrUpdate(ctx context.Context, org, name, plaintext, visibility string, selectedRepoIDs []int64) error {
	pubKey, _, err := p.client.Actions.GetOrgPublicKey(ctx, org)
	if err != nil {
		return fmt.Errorf("failed to get org public key: %w", err)
	}

	encrypted, err := crypto.EncryptSecret(plaintext, pubKey.GetKey())
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	eSecret := &github.EncryptedSecret{
		Name:           name,
		KeyID:          pubKey.GetKeyID(),
		EncryptedValue: encrypted,
		Visibility:     visibility,
	}
	if len(selectedRepoIDs) > 0 {
		eSecret.SelectedRepositoryIDs = github.SelectedRepoIDs(selectedRepoIDs)
	}

	_, err = p.client.Actions.CreateOrUpdateOrgSecret(ctx, org, eSecret)
	return err
}
