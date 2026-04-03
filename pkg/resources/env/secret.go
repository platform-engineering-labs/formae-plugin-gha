// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/go-github/v69/github"

	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/config"
	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/crypto"
	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/provisioner"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

const SecretResourceType = "GHA::Environment::Secret"

func init() {
	provisioner.Register(SecretResourceType, func(client *github.Client, cfg *config.Config) provisioner.Provisioner {
		return &envSecretProvisioner{client: client, cfg: cfg}
	})
}

type envSecretProperties struct {
	Environment string `json:"environment"`
	Name        string `json:"name"`
	Value       string `json:"value,omitempty"`
	ValueHash   string `json:"valueHash,omitempty"`
}

type envSecretProvisioner struct {
	client *github.Client
	cfg    *config.Config
}

func (p *envSecretProvisioner) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	var props envSecretProperties
	if err := json.Unmarshal(req.Properties, &props); err != nil {
		return provisioner.CreateFailure(resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	if err := p.createOrUpdate(ctx, p.cfg.Owner, p.cfg.Repo, props.Environment, props.Name, props.Value); err != nil {
		return provisioner.CreateFailure(provisioner.ClassifyError(err), err.Error()), nil
	}

	nativeID := props.Environment + "/" + props.Name
	resultProps := provisioner.MustMarshal(envSecretProperties{
		Environment: props.Environment,
		Name:        props.Name,
		ValueHash:   crypto.HashSecret(props.Value),
	})
	return provisioner.CreateSuccess(nativeID, resultProps), nil
}

func (p *envSecretProvisioner) Read(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
	parts := strings.SplitN(req.NativeID, "/", 2)
	if len(parts) != 2 {
		return &resource.ReadResult{ResourceType: SecretResourceType, ErrorCode: resource.OperationErrorCodeInternalFailure}, nil
	}
	envName, name := parts[0], parts[1]

	repoID, err := p.getRepoID(ctx, p.cfg.Owner, p.cfg.Repo)
	if err != nil {
		return &resource.ReadResult{ResourceType: SecretResourceType, ErrorCode: provisioner.ClassifyError(err)}, nil
	}

	_, _, err = p.client.Actions.GetEnvSecret(ctx, repoID, envName, name)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.ReadNotFound(SecretResourceType), nil
		}
		return &resource.ReadResult{ResourceType: SecretResourceType, ErrorCode: provisioner.ClassifyError(err)}, nil
	}

	props := provisioner.MustMarshal(envSecretProperties{Environment: envName, Name: name})
	return provisioner.ReadSuccess(SecretResourceType, props), nil
}

func (p *envSecretProvisioner) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	parts := strings.SplitN(req.NativeID, "/", 2)
	if len(parts) != 2 {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInternalFailure, fmt.Sprintf("invalid native ID %q", req.NativeID)), nil
	}
	envName, name := parts[0], parts[1]

	var desired envSecretProperties
	if err := json.Unmarshal(req.DesiredProperties, &desired); err != nil {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInvalidRequest, err.Error()), nil
	}

	if err := p.createOrUpdate(ctx, p.cfg.Owner, p.cfg.Repo, envName, name, desired.Value); err != nil {
		return provisioner.UpdateFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(envSecretProperties{
		Environment: envName,
		Name:        name,
		ValueHash:   crypto.HashSecret(desired.Value),
	})
	return provisioner.UpdateSuccess(req.NativeID, resultProps), nil
}

func (p *envSecretProvisioner) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	parts := strings.SplitN(req.NativeID, "/", 2)
	if len(parts) != 2 {
		return provisioner.DeleteFailure(req.NativeID, resource.OperationErrorCodeInternalFailure, fmt.Sprintf("invalid native ID %q", req.NativeID)), nil
	}
	envName, name := parts[0], parts[1]

	repoID, err := p.getRepoID(ctx, p.cfg.Owner, p.cfg.Repo)
	if err != nil {
		return provisioner.DeleteFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	_, err = p.client.Actions.DeleteEnvSecret(ctx, repoID, envName, name)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.DeleteSuccess(req.NativeID), nil
		}
		return provisioner.DeleteFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}
	return provisioner.DeleteSuccess(req.NativeID), nil
}

func (p *envSecretProvisioner) List(_ context.Context, _ *resource.ListRequest) (*resource.ListResult, error) {
	return &resource.ListResult{NativeIDs: []string{}}, nil
}

func (p *envSecretProvisioner) Status(_ context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	return provisioner.StatusSuccess(req.NativeID), nil
}

// getRepoID resolves the integer repository ID from owner/repo.
// The GitHub env secrets API requires this instead of owner/repo strings.
func (p *envSecretProvisioner) getRepoID(ctx context.Context, owner, repo string) (int, error) {
	repoObj, _, err := p.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return 0, err
	}
	return int(repoObj.GetID()), nil
}

// createOrUpdate encrypts and sends the secret to the environment.
// Environment secrets use the env-specific public key via GetEnvPublicKey,
// which requires the integer repository ID.
func (p *envSecretProvisioner) createOrUpdate(ctx context.Context, owner, repo, envName, name, plaintext string) error {
	// Get the integer repo ID needed for GetEnvPublicKey
	repoObj, _, err := p.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}

	pubKey, _, err := p.client.Actions.GetEnvPublicKey(ctx, int(repoObj.GetID()), envName)
	if err != nil {
		return fmt.Errorf("failed to get env public key: %w", err)
	}

	encrypted, err := crypto.EncryptSecret(plaintext, pubKey.GetKey())
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	_, err = p.client.Actions.CreateOrUpdateEnvSecret(ctx, int(repoObj.GetID()), envName, &github.EncryptedSecret{
		Name:           name,
		KeyID:          pubKey.GetKeyID(),
		EncryptedValue: encrypted,
	})
	return err
}
