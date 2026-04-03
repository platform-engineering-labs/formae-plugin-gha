// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"

	"github.com/google/go-github/v69/github"

	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/config"
	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/provisioner"
	"github.com/platform-engineering-labs/formae/pkg/plugin"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"

	// Blank imports register provisioners via init() functions.
	_ "github.com/platform-engineering-labs/formae-plugin-gha/pkg/resources/env"
	_ "github.com/platform-engineering-labs/formae-plugin-gha/pkg/resources/org"
	_ "github.com/platform-engineering-labs/formae-plugin-gha/pkg/resources/repo"
)

// Plugin implements the Formae ResourcePlugin interface.
type Plugin struct{}

var _ plugin.ResourcePlugin = &Plugin{}

func (p *Plugin) RateLimit() plugin.RateLimitConfig {
	return plugin.RateLimitConfig{
		Scope:                            plugin.RateLimitScopeNamespace,
		MaxRequestsPerSecondForNamespace: 1,
	}
}

func (p *Plugin) DiscoveryFilters() []plugin.MatchFilter {
	return nil
}

func (p *Plugin) LabelConfig() plugin.LabelConfig {
	return plugin.LabelConfig{
		DefaultQuery: "$.name",
	}
}

// getProvisioner creates an authenticated GitHub client and returns
// the appropriate provisioner for the given resource type.
func (p *Plugin) getProvisioner(resourceType string, targetConfig []byte) (provisioner.Provisioner, error) {
	factory, ok := provisioner.Get(resourceType)
	if !ok {
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	cfg, err := config.FromTargetConfig(targetConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	gh := github.NewClient(nil).WithAuthToken(cfg.Token)
	return factory(gh, cfg), nil
}

func (p *Plugin) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	prov, err := p.getProvisioner(req.ResourceType, req.TargetConfig)
	if err != nil {
		return nil, err
	}
	return prov.Create(ctx, req)
}

func (p *Plugin) Read(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
	prov, err := p.getProvisioner(req.ResourceType, req.TargetConfig)
	if err != nil {
		return nil, err
	}
	return prov.Read(ctx, req)
}

func (p *Plugin) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	prov, err := p.getProvisioner(req.ResourceType, req.TargetConfig)
	if err != nil {
		return nil, err
	}
	return prov.Update(ctx, req)
}

func (p *Plugin) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	prov, err := p.getProvisioner(req.ResourceType, req.TargetConfig)
	if err != nil {
		return nil, err
	}
	return prov.Delete(ctx, req)
}

func (p *Plugin) Status(ctx context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	prov, err := p.getProvisioner(req.ResourceType, req.TargetConfig)
	if err != nil {
		return nil, err
	}
	return prov.Status(ctx, req)
}

func (p *Plugin) List(ctx context.Context, req *resource.ListRequest) (*resource.ListResult, error) {
	prov, err := p.getProvisioner(req.ResourceType, req.TargetConfig)
	if err != nil {
		return nil, err
	}
	return prov.List(ctx, req)
}
