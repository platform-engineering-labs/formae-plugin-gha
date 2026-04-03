// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds target configuration for the GHA plugin.
type Config struct {
	Owner string `json:"Owner"`
	Repo  string `json:"Repo,omitempty"`
	Token string `json:"-"`
}

// FromTargetConfig parses target configuration JSON and resolves the GitHub token
// via the auth chain: target config → GITHUB_TOKEN env → gh CLI → gh config file.
func FromTargetConfig(targetConfig []byte) (*Config, error) {
	cfg := &Config{}
	if len(targetConfig) > 0 {
		if err := json.Unmarshal(targetConfig, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse target config: %w", err)
		}
	}

	cfg.Token = resolveToken()

	return cfg, nil
}

// Validate checks that all required fields are present.
func (c *Config) Validate() error {
	if c.Token == "" {
		return fmt.Errorf("GitHub token not found. Checked: GITHUB_TOKEN env, gh auth token, ~/.config/gh/hosts.yml")
	}
	if c.Owner == "" {
		return fmt.Errorf("Owner is required in target config")
	}
	return nil
}

// resolveToken tries multiple sources to find a GitHub token.
// Order: GITHUB_TOKEN env → gh auth token CLI → gh config file.
func resolveToken() string {
	// 1. Environment variable
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token
	}

	// 2. gh auth token command
	if token := ghAuthToken(); token != "" {
		return token
	}

	// 3. gh config file
	if token := ghConfigToken(); token != "" {
		return token
	}

	return ""
}

// ghAuthToken runs `gh auth token` and returns the token if successful.
func ghAuthToken() string {
	out, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ghConfigToken reads the token from ~/.config/gh/hosts.yml.
func ghConfigToken() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	data, err := os.ReadFile(home + "/.config/gh/hosts.yml")
	if err != nil {
		return ""
	}

	var hosts map[string]struct {
		OAuthToken string `yaml:"oauth_token"`
	}
	if err := yaml.Unmarshal(data, &hosts); err != nil {
		return ""
	}

	if host, ok := hosts["github.com"]; ok {
		return host.OAuthToken
	}

	return ""
}
