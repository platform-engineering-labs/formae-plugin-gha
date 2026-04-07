// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: Apache-2.0

package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/google/go-github/v69/github"

	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/config"
	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/provisioner"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

const WorkflowResourceType = "GHA::Repo::Workflow"

// workflowKeyOrder defines the conventional key ordering for GitHub Actions workflow YAML.
var workflowKeyOrder = []string{"name", "run-name", "on", "permissions", "env", "defaults", "concurrency", "jobs"}

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

	delete(props, "path")

	yamlContent, err := marshalWorkflowYAML(props)
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

	yamlContent, err := marshalWorkflowYAML(props)
	if err != nil {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInternalFailure, fmt.Sprintf("failed to marshal YAML: %v", err)), nil
	}

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

// marshalWorkflowYAML converts a workflow properties map to YAML with conventional
// key ordering (name, on, permissions, env, jobs) using yaml.MapSlice to preserve
// insertion order.
func marshalWorkflowYAML(data map[string]interface{}) ([]byte, error) {
	var ms yaml.MapSlice

	// Add keys in conventional workflow order
	for _, key := range workflowKeyOrder {
		val, ok := data[key]
		if !ok {
			continue
		}
		ms = append(ms, yaml.MapItem{Key: key, Value: val})
	}

	// Add any remaining keys not in the order list
	for key, val := range data {
		inOrder := false
		for _, k := range workflowKeyOrder {
			if key == k {
				inOrder = true
				break
			}
		}
		if inOrder {
			continue
		}
		ms = append(ms, yaml.MapItem{Key: key, Value: val})
	}

	return yaml.Marshal(ms)
}
