// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package provisioner

import (
	"fmt"
	"sync"

	"github.com/google/go-github/v69/github"

	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/config"
)

// Factory creates a Provisioner given an authenticated GitHub client and config.
type Factory func(client *github.Client, cfg *config.Config) Provisioner

var (
	mu        sync.RWMutex
	factories = make(map[string]Factory)
)

// Register associates a resource type with its provisioner factory.
// Called from init() functions in resource packages.
func Register(resourceType string, factory Factory) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := factories[resourceType]; exists {
		panic(fmt.Sprintf("provisioner already registered for %s", resourceType))
	}
	factories[resourceType] = factory
}

// Get returns the factory for a resource type, or false if not registered.
func Get(resourceType string) (Factory, bool) {
	mu.RLock()
	defer mu.RUnlock()
	f, ok := factories[resourceType]
	return f, ok
}
