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
	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/crypto"
	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/provisioner"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

const SecretResourceType = "GHA::Repo::Secret"

func init() {
	provisioner.Register(SecretResourceType, func(client *github.Client, cfg *config.Config) provisioner.Provisioner {
		return &secretProvisioner{client: client, cfg: cfg}
	})
}

type secretProperties struct {
	Name      string `json:"name"`
	Value     string `json:"value,omitempty"`
	ValueHash string `json:"valueHash,omitempty"`
}

type secretProvisioner struct {
	client *github.Client
	cfg    *config.Config
}

func (p *secretProvisioner) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	var props secretProperties
	if err := json.Unmarshal(req.Properties, &props); err != nil {
		return provisioner.CreateFailure(resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	if err := p.createOrUpdate(ctx, p.cfg.Owner, p.cfg.Repo, props.Name, props.Value); err != nil {
		return provisioner.CreateFailure(provisioner.ClassifyError(err), err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(secretProperties{
		Name:      props.Name,
		ValueHash: crypto.HashSecret(props.Value),
	})
	return provisioner.CreateSuccess(props.Name, resultProps), nil
}

func (p *secretProvisioner) Read(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
	_, _, err := p.client.Actions.GetRepoSecret(ctx, p.cfg.Owner, p.cfg.Repo, req.NativeID)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.ReadNotFound(SecretResourceType), nil
		}
		return &resource.ReadResult{ResourceType: SecretResourceType, ErrorCode: provisioner.ClassifyError(err)}, nil
	}

	props := provisioner.MustMarshal(secretProperties{Name: req.NativeID})
	return provisioner.ReadSuccess(SecretResourceType, props), nil
}

func (p *secretProvisioner) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var desired secretProperties
	if err := json.Unmarshal(req.DesiredProperties, &desired); err != nil {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInvalidRequest, err.Error()), nil
	}

	if err := p.createOrUpdate(ctx, p.cfg.Owner, p.cfg.Repo, req.NativeID, desired.Value); err != nil {
		return provisioner.UpdateFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(secretProperties{
		Name:      req.NativeID,
		ValueHash: crypto.HashSecret(desired.Value),
	})
	return provisioner.UpdateSuccess(req.NativeID, resultProps), nil
}

func (p *secretProvisioner) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	_, err := p.client.Actions.DeleteRepoSecret(ctx, p.cfg.Owner, p.cfg.Repo, req.NativeID)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.DeleteSuccess(req.NativeID), nil
		}
		return provisioner.DeleteFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	return provisioner.DeleteSuccess(req.NativeID), nil
}

func (p *secretProvisioner) List(ctx context.Context, req *resource.ListRequest) (*resource.ListResult, error) {
	var allIDs []string
	opts := &github.ListOptions{PerPage: 100}
	for {
		secrets, resp, err := p.client.Actions.ListRepoSecrets(ctx, p.cfg.Owner, p.cfg.Repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list repo secrets: %w", err)
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

func (p *secretProvisioner) Status(_ context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	return provisioner.StatusSuccess(req.NativeID), nil
}

func (p *secretProvisioner) createOrUpdate(ctx context.Context, owner, repo, name, plaintext string) error {
	pubKey, _, err := p.client.Actions.GetRepoPublicKey(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to get public key: %w", err)
	}

	encrypted, err := crypto.EncryptSecret(plaintext, pubKey.GetKey())
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	_, err = p.client.Actions.CreateOrUpdateRepoSecret(ctx, owner, repo, &github.EncryptedSecret{
		Name:           name,
		KeyID:          pubKey.GetKeyID(),
		EncryptedValue: encrypted,
	})
	return err
}
