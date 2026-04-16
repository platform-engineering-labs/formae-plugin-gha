// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: Apache-2.0

//go:build unit

package repo

import (
	"strings"
	"testing"
)

// topLevelKeyPos returns the position of a top-level YAML key in the output.
// Handles both unquoted (name:) and quoted ("on":) forms.
func topLevelKeyPos(yaml, key string) int {
	// Try unquoted at start of line
	needle := "\n" + key + ":"
	if idx := strings.Index(yaml, needle); idx != -1 {
		return idx
	}
	// Try at very start of string
	if strings.HasPrefix(yaml, key+":") {
		return 0
	}
	// Try quoted form (e.g. "on":)
	needle = "\n\"" + key + "\":"
	if idx := strings.Index(yaml, needle); idx != -1 {
		return idx
	}
	if strings.HasPrefix(yaml, "\""+key+"\":") {
		return 0
	}
	return -1
}

func TestMarshalWorkflowYAML_KeyOrdering(t *testing.T) {
	data := map[string]interface{}{
		"jobs": map[string]interface{}{
			"build": map[string]interface{}{
				"runs-on": "ubuntu-latest",
				"steps": []interface{}{
					map[string]interface{}{
						"name": "Checkout",
						"uses": "actions/checkout@v4",
					},
				},
			},
		},
		"on": map[string]interface{}{
			"push": map[string]interface{}{
				"branches": []interface{}{"main"},
			},
		},
		"name": "CI",
		"permissions": map[string]interface{}{
			"contents": "read",
		},
		"env": map[string]interface{}{
			"GO_VERSION": "1.25",
		},
	}

	out, err := marshalWorkflowYAML(data)
	if err != nil {
		t.Fatalf("marshalWorkflowYAML error: %v", err)
	}

	yaml := string(out)

	// Verify conventional key ordering: name < on < permissions < env < jobs
	nameIdx := topLevelKeyPos(yaml, "name")
	onIdx := topLevelKeyPos(yaml, "on")
	permIdx := topLevelKeyPos(yaml, "permissions")
	envIdx := topLevelKeyPos(yaml, "env")
	jobsIdx := topLevelKeyPos(yaml, "jobs")

	if nameIdx == -1 || onIdx == -1 || permIdx == -1 || envIdx == -1 || jobsIdx == -1 {
		t.Fatalf("missing expected top-level keys in output:\n%s", yaml)
	}

	if nameIdx >= onIdx {
		t.Errorf("'name' should appear before 'on' in output:\n%s", yaml)
	}
	if onIdx >= permIdx {
		t.Errorf("'on' should appear before 'permissions' in output:\n%s", yaml)
	}
	if permIdx >= envIdx {
		t.Errorf("'permissions' should appear before 'env' in output:\n%s", yaml)
	}
	if envIdx >= jobsIdx {
		t.Errorf("'env' should appear before 'jobs' in output:\n%s", yaml)
	}
}

func TestMarshalWorkflowYAML_AllPropertiesPreserved(t *testing.T) {
	data := map[string]interface{}{
		"name": "Deploy",
		"on": map[string]interface{}{
			"push": map[string]interface{}{
				"branches": []interface{}{"main", "release/*"},
			},
			"pull_request": map[string]interface{}{
				"types": []interface{}{"opened", "synchronize"},
			},
		},
		"permissions": map[string]interface{}{
			"contents":    "read",
			"id-token":    "write",
			"deployments": "write",
		},
		"env": map[string]interface{}{
			"REGION":      "us-east-1",
			"GO_VERSION":  "1.25",
			"ENVIRONMENT": "production",
		},
		"concurrency": map[string]interface{}{
			"group":            "deploy-${{ github.ref }}",
			"cancel-in-progress": true,
		},
		"jobs": map[string]interface{}{
			"build": map[string]interface{}{
				"runs-on": "ubuntu-latest",
				"steps": []interface{}{
					map[string]interface{}{
						"name": "Checkout",
						"uses": "actions/checkout@v4",
					},
					map[string]interface{}{
						"name": "Build",
						"run":  "make build",
					},
				},
			},
			"deploy": map[string]interface{}{
				"runs-on": "ubuntu-latest",
				"needs":   "build",
				"steps": []interface{}{
					map[string]interface{}{
						"name": "Deploy",
						"run":  "make deploy",
					},
				},
			},
		},
	}

	out, err := marshalWorkflowYAML(data)
	if err != nil {
		t.Fatalf("marshalWorkflowYAML error: %v", err)
	}

	yaml := string(out)

	// Verify all values are present in output
	requiredStrings := []string{
		"Deploy",
		"main",
		"release/*",
		"opened",
		"synchronize",
		"contents",
		"id-token",
		"deployments",
		"REGION",
		"us-east-1",
		"GO_VERSION",
		"ENVIRONMENT",
		"production",
		"cancel-in-progress",
		"build",
		"deploy",
		"ubuntu-latest",
		"actions/checkout@v4",
		"make build",
		"make deploy",
	}

	for _, s := range requiredStrings {
		if !strings.Contains(yaml, s) {
			t.Errorf("output missing %q:\n%s", s, yaml)
		}
	}
}

func TestMarshalWorkflowYAML_NonStandardKeysPreserved(t *testing.T) {
	data := map[string]interface{}{
		"name":         "CI",
		"on":           "push",
		"custom-field": "custom-value",
		"another-key":  42,
		"jobs": map[string]interface{}{
			"test": map[string]interface{}{
				"runs-on": "ubuntu-latest",
			},
		},
	}

	out, err := marshalWorkflowYAML(data)
	if err != nil {
		t.Fatalf("marshalWorkflowYAML error: %v", err)
	}

	yaml := string(out)

	if !strings.Contains(yaml, "custom-field") {
		t.Errorf("output missing non-standard key 'custom-field':\n%s", yaml)
	}
	if !strings.Contains(yaml, "custom-value") {
		t.Errorf("output missing non-standard value 'custom-value':\n%s", yaml)
	}
	if !strings.Contains(yaml, "another-key") {
		t.Errorf("output missing non-standard key 'another-key':\n%s", yaml)
	}
	if !strings.Contains(yaml, "42") {
		t.Errorf("output missing non-standard value '42':\n%s", yaml)
	}

	// Standard keys should still come first
	nameIdx := strings.Index(yaml, "name:")
	jobsIdx := strings.Index(yaml, "jobs:")
	customIdx := strings.Index(yaml, "custom-field:")

	if nameIdx >= jobsIdx {
		t.Errorf("'name' should appear before 'jobs':\n%s", yaml)
	}
	if customIdx <= jobsIdx {
		t.Errorf("non-standard 'custom-field' should appear after standard keys:\n%s", yaml)
	}
}
