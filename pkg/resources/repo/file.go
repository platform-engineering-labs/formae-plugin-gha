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

const FileResourceType = "GHA::Repo::File"

func init() {
	provisioner.Register(FileResourceType, func(client *github.Client, cfg *config.Config) provisioner.Provisioner {
		return &fileProvisioner{client: client, cfg: cfg}
	})
}

type fileProperties struct {
	Path          string `json:"path"`
	Content       string `json:"content"`
	CommitMessage string `json:"commitMessage,omitempty"`
	Branch        string `json:"branch,omitempty"`
	SHA           string `json:"sha,omitempty"`
}

type fileProvisioner struct {
	client *github.Client
	cfg    *config.Config
}

func (p *fileProvisioner) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	var props fileProperties
	if err := json.Unmarshal(req.Properties, &props); err != nil {
		return provisioner.CreateFailure(resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	commitMsg := props.CommitMessage
	if commitMsg == "" {
		commitMsg = "managed by formae"
	}

	opts := &github.RepositoryContentFileOptions{
		Message: github.Ptr(commitMsg),
		Content: []byte(props.Content),
	}
	if props.Branch != "" {
		opts.Branch = github.Ptr(props.Branch)
	}

	resp, _, err := p.client.Repositories.CreateFile(ctx, p.cfg.Owner, p.cfg.Repo, props.Path, opts)
	if err != nil {
		return provisioner.CreateFailure(provisioner.ClassifyError(err), err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(fileProperties{
		Path:          props.Path,
		Content:       props.Content,
		CommitMessage: commitMsg,
		Branch:        props.Branch,
		SHA:           resp.Content.GetSHA(),
	})
	return provisioner.CreateSuccess(props.Path, resultProps), nil
}

func (p *fileProvisioner) Read(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
	fileContent, _, _, err := p.client.Repositories.GetContents(ctx, p.cfg.Owner, p.cfg.Repo, req.NativeID, nil)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.ReadNotFound(FileResourceType), nil
		}
		return &resource.ReadResult{ResourceType: FileResourceType, ErrorCode: provisioner.ClassifyError(err)}, nil
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return &resource.ReadResult{ResourceType: FileResourceType, ErrorCode: resource.OperationErrorCodeInternalFailure}, nil
	}

	props := provisioner.MustMarshal(fileProperties{
		Path:    req.NativeID,
		Content: content,
		SHA:     fileContent.GetSHA(),
	})
	return provisioner.ReadSuccess(FileResourceType, props), nil
}

func (p *fileProvisioner) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var desired fileProperties
	if err := json.Unmarshal(req.DesiredProperties, &desired); err != nil {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInvalidRequest, err.Error()), nil
	}

	currentFile, _, _, err := p.client.Repositories.GetContents(ctx, p.cfg.Owner, p.cfg.Repo, req.NativeID, nil)
	if err != nil {
		return provisioner.UpdateFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	commitMsg := desired.CommitMessage
	if commitMsg == "" {
		commitMsg = "managed by formae"
	}

	opts := &github.RepositoryContentFileOptions{
		Message: github.Ptr(commitMsg),
		Content: []byte(desired.Content),
		SHA:     currentFile.SHA,
	}
	if desired.Branch != "" {
		opts.Branch = github.Ptr(desired.Branch)
	}

	resp, _, err := p.client.Repositories.UpdateFile(ctx, p.cfg.Owner, p.cfg.Repo, req.NativeID, opts)
	if err != nil {
		return provisioner.UpdateFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	resultProps := provisioner.MustMarshal(fileProperties{
		Path:          req.NativeID,
		Content:       desired.Content,
		CommitMessage: commitMsg,
		Branch:        desired.Branch,
		SHA:           resp.Content.GetSHA(),
	})
	return provisioner.UpdateSuccess(req.NativeID, resultProps), nil
}

func (p *fileProvisioner) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	currentFile, _, _, err := p.client.Repositories.GetContents(ctx, p.cfg.Owner, p.cfg.Repo, req.NativeID, nil)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.DeleteSuccess(req.NativeID), nil
		}
		return provisioner.DeleteFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	_, _, err = p.client.Repositories.DeleteFile(ctx, p.cfg.Owner, p.cfg.Repo, req.NativeID, &github.RepositoryContentFileOptions{
		Message: github.Ptr("removed by formae"),
		SHA:     currentFile.SHA,
	})
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.DeleteSuccess(req.NativeID), nil
		}
		return provisioner.DeleteFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	return provisioner.DeleteSuccess(req.NativeID), nil
}

func (p *fileProvisioner) List(ctx context.Context, req *resource.ListRequest) (*resource.ListResult, error) {
	_, dirContent, _, err := p.client.Repositories.GetContents(ctx, p.cfg.Owner, p.cfg.Repo, ".github/workflows", nil)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return &resource.ListResult{NativeIDs: []string{}}, nil
		}
		return nil, fmt.Errorf("failed to list repo files: %w", err)
	}

	var allIDs []string
	for _, f := range dirContent {
		if f.GetType() == "file" {
			allIDs = append(allIDs, f.GetPath())
		}
	}

	return &resource.ListResult{NativeIDs: allIDs}, nil
}

func (p *fileProvisioner) Status(_ context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	return provisioner.StatusSuccess(req.NativeID), nil
}
