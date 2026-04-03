// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: Apache-2.0

package provisioner

import (
	"context"

	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

// Provisioner defines the CRUD + List + Status operations for a resource type.
type Provisioner interface {
	Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error)
	Read(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error)
	Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error)
	Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error)
	List(ctx context.Context, req *resource.ListRequest) (*resource.ListResult, error)
	Status(ctx context.Context, req *resource.StatusRequest) (*resource.StatusResult, error)
}
