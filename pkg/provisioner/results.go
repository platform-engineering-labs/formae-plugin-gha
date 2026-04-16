// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package provisioner

import (
	"encoding/json"
	"fmt"

	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

// MustMarshal marshals v to JSON or panics. Use for internal structs
// where marshal failure indicates a programming error.
func MustMarshal(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("provisioner: failed to marshal %T: %v", v, err))
	}
	return data
}

// CreateSuccess returns a successful create result.
func CreateSuccess(nativeID string, properties json.RawMessage) *resource.CreateResult {
	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCreate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           nativeID,
			ResourceProperties: properties,
		},
	}
}

// CreateFailure returns a failed create result.
func CreateFailure(errorCode resource.OperationErrorCode, msg string) *resource.CreateResult {
	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCreate,
			OperationStatus: resource.OperationStatusFailure,
			ErrorCode:       errorCode,
			StatusMessage:   msg,
		},
	}
}

// ReadSuccess returns a successful read result.
func ReadSuccess(resourceType string, properties json.RawMessage) *resource.ReadResult {
	return &resource.ReadResult{
		ResourceType: resourceType,
		Properties:   string(properties),
	}
}

// ReadNotFound returns a read result indicating the resource was not found.
func ReadNotFound(resourceType string) *resource.ReadResult {
	return &resource.ReadResult{
		ResourceType: resourceType,
		Properties:   "",
		ErrorCode:    resource.OperationErrorCodeNotFound,
	}
}

// UpdateSuccess returns a successful update result.
func UpdateSuccess(nativeID string, properties json.RawMessage) *resource.UpdateResult {
	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationUpdate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           nativeID,
			ResourceProperties: properties,
		},
	}
}

// UpdateFailure returns a failed update result.
func UpdateFailure(nativeID string, errorCode resource.OperationErrorCode, msg string) *resource.UpdateResult {
	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationUpdate,
			OperationStatus: resource.OperationStatusFailure,
			NativeID:        nativeID,
			ErrorCode:       errorCode,
			StatusMessage:   msg,
		},
	}
}

// DeleteSuccess returns a successful delete result.
func DeleteSuccess(nativeID string) *resource.DeleteResult {
	return &resource.DeleteResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationDelete,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        nativeID,
		},
	}
}

// DeleteFailure returns a failed delete result.
func DeleteFailure(nativeID string, errorCode resource.OperationErrorCode, msg string) *resource.DeleteResult {
	return &resource.DeleteResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationDelete,
			OperationStatus: resource.OperationStatusFailure,
			NativeID:        nativeID,
			ErrorCode:       errorCode,
			StatusMessage:   msg,
		},
	}
}

// StatusSuccess returns a synchronous status success result.
// Used when all operations are synchronous (no polling needed).
func StatusSuccess(nativeID string) *resource.StatusResult {
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        nativeID,
		},
	}
}
