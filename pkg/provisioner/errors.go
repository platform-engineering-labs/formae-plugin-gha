// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package provisioner

import (
	"errors"
	"net/http"

	"github.com/google/go-github/v69/github"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

// ClassifyError maps a GitHub API error to a formae OperationErrorCode.
func ClassifyError(err error) resource.OperationErrorCode {
	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) {
		return classifyHTTPStatus(ghErr.Response.StatusCode)
	}
	var rateLimitErr *github.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return resource.OperationErrorCodeThrottling
	}
	var abuseErr *github.AbuseRateLimitError
	if errors.As(err, &abuseErr) {
		return resource.OperationErrorCodeThrottling
	}
	return resource.OperationErrorCodeInternalFailure
}

func classifyHTTPStatus(status int) resource.OperationErrorCode {
	switch status {
	case http.StatusNotFound:
		return resource.OperationErrorCodeNotFound
	case http.StatusConflict:
		return resource.OperationErrorCodeAlreadyExists
	case http.StatusUnauthorized:
		return resource.OperationErrorCodeInvalidCredentials
	case http.StatusForbidden:
		return resource.OperationErrorCodeAccessDenied
	case http.StatusUnprocessableEntity:
		return resource.OperationErrorCodeInvalidRequest
	case http.StatusTooManyRequests:
		return resource.OperationErrorCodeThrottling
	default:
		if status >= 500 {
			return resource.OperationErrorCodeServiceInternalError
		}
		return resource.OperationErrorCodeInternalFailure
	}
}

// IsNotFound returns true if the error is a GitHub 404 response.
func IsNotFound(err error) bool {
	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) {
		return ghErr.Response.StatusCode == http.StatusNotFound
	}
	return false
}
