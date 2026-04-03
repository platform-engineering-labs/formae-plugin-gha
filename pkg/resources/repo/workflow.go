// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: Apache-2.0

package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/go-github/v69/github"
	"gopkg.in/yaml.v3"

	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/config"
	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/provisioner"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

const WorkflowResourceType = "GHA::Repo::Workflow"

func init() {
	provisioner.Register(WorkflowResourceType, func(client *github.Client, cfg *config.Config) provisioner.Provisioner {
		return &workflowProvisioner{client: client, cfg: cfg}
	})
}

type workflowProvisioner struct {
	client *github.Client
	cfg    *config.Config
}

func (p *workflowProvisioner) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	var props map[string]interface{}
	if err := json.Unmarshal(req.Properties, &props); err != nil {
		return provisioner.CreateFailure(resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	path, ok := props["path"].(string)
	if !ok || path == "" {
		return provisioner.CreateFailure(resource.OperationErrorCodeInvalidRequest, "path is required"), nil
	}

	// Remove path and empty fields — path isn't part of the YAML, and empty
	// maps/nulls would create false drift if GitHub strips them on read.
	delete(props, "path")
	stripEmpty(props)

	yamlContent, err := yaml.Marshal(props)
	if err != nil {
		return provisioner.CreateFailure(resource.OperationErrorCodeInternalFailure, fmt.Sprintf("failed to marshal YAML: %v", err)), nil
	}

	opts := &github.RepositoryContentFileOptions{
		Message: github.Ptr("managed by formae"),
		Content: yamlContent,
	}

	_, _, err = p.client.Repositories.CreateFile(ctx, p.cfg.Owner, p.cfg.Repo, path, opts)
	if err != nil {
		return provisioner.CreateFailure(provisioner.ClassifyError(err), err.Error()), nil
	}

	// Return original properties (including path).
	return provisioner.CreateSuccess(path, req.Properties), nil
}

func (p *workflowProvisioner) Read(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
	fileContent, _, _, err := p.client.Repositories.GetContents(ctx, p.cfg.Owner, p.cfg.Repo, req.NativeID, nil)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.ReadNotFound(WorkflowResourceType), nil
		}
		return &resource.ReadResult{ResourceType: WorkflowResourceType, ErrorCode: provisioner.ClassifyError(err)}, nil
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return &resource.ReadResult{ResourceType: WorkflowResourceType, ErrorCode: resource.OperationErrorCodeInternalFailure}, nil
	}

	var workflowMap map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &workflowMap); err != nil {
		return &resource.ReadResult{ResourceType: WorkflowResourceType, ErrorCode: resource.OperationErrorCodeInternalFailure}, nil
	}

	// Add path back to the structured properties.
	workflowMap["path"] = req.NativeID

	props := provisioner.MustMarshal(workflowMap)
	return provisioner.ReadSuccess(WorkflowResourceType, props), nil
}

func (p *workflowProvisioner) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var props map[string]interface{}
	if err := json.Unmarshal(req.DesiredProperties, &props); err != nil {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	delete(props, "path")
	stripEmpty(props)

	yamlContent, err := yaml.Marshal(props)
	if err != nil {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInternalFailure, fmt.Sprintf("failed to marshal YAML: %v", err)), nil
	}

	// Get current SHA for the update.
	currentFile, _, _, err := p.client.Repositories.GetContents(ctx, p.cfg.Owner, p.cfg.Repo, req.NativeID, nil)
	if err != nil {
		return provisioner.UpdateFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	opts := &github.RepositoryContentFileOptions{
		Message: github.Ptr("managed by formae"),
		Content: yamlContent,
		SHA:     currentFile.SHA,
	}

	_, _, err = p.client.Repositories.UpdateFile(ctx, p.cfg.Owner, p.cfg.Repo, req.NativeID, opts)
	if err != nil {
		return provisioner.UpdateFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	return provisioner.UpdateSuccess(req.NativeID, req.DesiredProperties), nil
}

func (p *workflowProvisioner) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
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

func (p *workflowProvisioner) List(ctx context.Context, req *resource.ListRequest) (*resource.ListResult, error) {
	_, dirContent, _, err := p.client.Repositories.GetContents(ctx, p.cfg.Owner, p.cfg.Repo, ".github/workflows", nil)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return &resource.ListResult{NativeIDs: []string{}}, nil
		}
		return nil, fmt.Errorf("failed to list repo workflows: %w", err)
	}

	var allIDs []string
	for _, f := range dirContent {
		if f.GetType() == "file" {
			allIDs = append(allIDs, f.GetPath())
		}
	}

	return &resource.ListResult{NativeIDs: allIDs}, nil
}

func (p *workflowProvisioner) Status(_ context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	return provisioner.StatusSuccess(req.NativeID), nil
}

// stripEmpty recursively removes nil values and empty maps from a map.
func stripEmpty(m map[string]interface{}) {
	for k, v := range m {
		switch val := v.(type) {
		case nil:
			delete(m, k)
		case map[string]interface{}:
			stripEmpty(val)
			if len(val) == 0 {
				delete(m, k)
			}
		}
	}
}
