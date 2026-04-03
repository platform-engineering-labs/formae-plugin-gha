#!/bin/bash
# © 2025 Platform Engineering Labs Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Clean Environment Hook for GHA Plugin
# ======================================
# Called before AND after conformance tests to clean up test resources.
# Idempotent — safe to run multiple times.

set -uo pipefail

OWNER="${GHA_TEST_OWNER:-platform-engineering-labs}"
REPO="${GHA_TEST_REPO:-formae-plugin-gha-test}"
PREFIX="FORMAE_TEST_"

echo "clean-environment.sh: Cleaning GHA test resources on ${OWNER}/${REPO}"

# Clean env variables (must happen before environments are deleted)
echo "Cleaning environment variables..."
gh api "repos/${OWNER}/${REPO}/environments" --jq '.environments[].name' 2>/dev/null | while read -r env; do
    gh api "repos/${OWNER}/${REPO}/environments/${env}/variables" --jq '.variables[].name' 2>/dev/null | while read -r name; do
        echo "  Deleting env variable: ${env}/${name}"
        gh api -X DELETE "repos/${OWNER}/${REPO}/environments/${env}/variables/${name}" 2>/dev/null || true
    done
    gh api "repos/${OWNER}/${REPO}/environments/${env}/secrets" --jq '.secrets[].name' 2>/dev/null | while read -r name; do
        echo "  Deleting env secret: ${env}/${name}"
        gh api -X DELETE "repos/${OWNER}/${REPO}/environments/${env}/secrets/${name}" 2>/dev/null || true
    done
done

# Clean repo variables
echo "Cleaning repo variables..."
gh api "repos/${OWNER}/${REPO}/actions/variables" --paginate --jq '.variables[].name' 2>/dev/null | while read -r name; do
    echo "  Deleting variable: ${name}"
    gh api -X DELETE "repos/${OWNER}/${REPO}/actions/variables/${name}" 2>/dev/null || true
done

# Clean repo secrets
echo "Cleaning repo secrets..."
gh api "repos/${OWNER}/${REPO}/actions/secrets" --paginate --jq '.secrets[].name' 2>/dev/null | while read -r name; do
    echo "  Deleting secret: ${name}"
    gh api -X DELETE "repos/${OWNER}/${REPO}/actions/secrets/${name}" 2>/dev/null || true
done

# Clean environments
echo "Cleaning environments..."
gh api "repos/${OWNER}/${REPO}/environments" --jq '.environments[].name' 2>/dev/null | while read -r name; do
        echo "  Deleting environment: ${name}"
        gh api -X DELETE "repos/${OWNER}/${REPO}/environments/${name}" 2>/dev/null || true
    done

# Clean test files
echo "Cleaning test files..."
for path in test .github/workflows; do
    gh api "repos/${OWNER}/${REPO}/contents/${path}" --jq '.[].path' 2>/dev/null | \
        grep -i "formae\|test\|inttest" | while read -r filepath; do
            sha=$(gh api "repos/${OWNER}/${REPO}/contents/${filepath}" --jq '.sha' 2>/dev/null || echo "")
            if [ -n "$sha" ]; then
                echo "  Deleting file: ${filepath}"
                gh api -X DELETE "repos/${OWNER}/${REPO}/contents/${filepath}" \
                    -f message="test: cleanup" -f sha="${sha}" 2>/dev/null || true
            fi
        done
done

echo "clean-environment.sh: Cleanup complete"
