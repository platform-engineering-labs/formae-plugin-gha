// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: Apache-2.0

//go:build integration

package org

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/go-github/v69/github"

	"github.com/platform-engineering-labs/formae-plugin-gha/pkg/config"
)

func testClient(t *testing.T) *github.Client {
	t.Helper()
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("GITHUB_TOKEN not set")
	}
	return github.NewClient(nil).WithAuthToken(token)
}

func testOwner(t *testing.T) string {
	t.Helper()
	owner := os.Getenv("GHA_TEST_OWNER")
	if owner == "" {
		t.Skip("GHA_TEST_OWNER not set")
	}
	return owner
}

func testTargetConfig(t *testing.T) []byte {
	t.Helper()
	owner := testOwner(t)
	cfg, _ := json.Marshal(map[string]string{"Owner": owner})
	return cfg
}

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{Owner: testOwner(t)}
}

func uniqueName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano()%100000)
}
