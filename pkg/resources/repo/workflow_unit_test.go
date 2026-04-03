// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: Apache-2.0

//go:build unit

package repo

import (
	"testing"
)

func TestStripEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:     "nil value removed",
			input:    map[string]interface{}{"a": "keep", "b": nil},
			expected: map[string]interface{}{"a": "keep"},
		},
		{
			name:     "empty map removed",
			input:    map[string]interface{}{"a": "keep", "env": map[string]interface{}{}},
			expected: map[string]interface{}{"a": "keep"},
		},
		{
			name:     "nested empty map removed",
			input:    map[string]interface{}{"outer": map[string]interface{}{"inner": map[string]interface{}{}}},
			expected: map[string]interface{}{},
		},
		{
			name:     "non-empty map preserved",
			input:    map[string]interface{}{"env": map[string]interface{}{"FOO": "bar"}},
			expected: map[string]interface{}{"env": map[string]interface{}{"FOO": "bar"}},
		},
		{
			name:     "scalars preserved",
			input:    map[string]interface{}{"name": "test", "count": 42, "enabled": true},
			expected: map[string]interface{}{"name": "test", "count": 42, "enabled": true},
		},
		{
			name:     "arrays preserved",
			input:    map[string]interface{}{"steps": []interface{}{"a", "b"}},
			expected: map[string]interface{}{"steps": []interface{}{"a", "b"}},
		},
		{
			name:     "empty array preserved",
			input:    map[string]interface{}{"items": []interface{}{}},
			expected: map[string]interface{}{"items": []interface{}{}},
		},
		{
			name: "mixed deep structure",
			input: map[string]interface{}{
				"name": "deploy",
				"env":  map[string]interface{}{},
				"on":   map[string]interface{}{"push": map[string]interface{}{"branches": []interface{}{"main"}}},
				"jobs": map[string]interface{}{
					"build": map[string]interface{}{
						"runs-on": "ubuntu-latest",
						"env":     map[string]interface{}{},
						"defaults": nil,
					},
				},
			},
			expected: map[string]interface{}{
				"name": "deploy",
				"on":   map[string]interface{}{"push": map[string]interface{}{"branches": []interface{}{"main"}}},
				"jobs": map[string]interface{}{
					"build": map[string]interface{}{
						"runs-on": "ubuntu-latest",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stripEmpty(tt.input)
			if !mapsEqual(tt.input, tt.expected) {
				t.Errorf("got %v, want %v", tt.input, tt.expected)
			}
		})
	}
}

func mapsEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		switch va := va.(type) {
		case map[string]interface{}:
			vb, ok := vb.(map[string]interface{})
			if !ok || !mapsEqual(va, vb) {
				return false
			}
		case []interface{}:
			vb, ok := vb.([]interface{})
			if !ok || len(va) != len(vb) {
				return false
			}
		default:
			if va != vb {
				return false
			}
		}
	}
	return true
}
