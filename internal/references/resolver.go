// SPDX-License-Identifier: Apache-2.0

// Package references contains safe managed-resource reference resolution.
package references

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewResolver persists a reference-resolution delta using a merge patch, not
// server-side apply. Applying only a delta relinquishes ownership of previously
// resolved fields and can delete them on the next resolution pass.
func NewResolver(c client.Client) managed.ReferenceResolver {
	return managed.ReferenceResolverFn(func(ctx context.Context, mg resource.Managed) error {
		rr, ok := mg.(interface {
			ResolveReferences(context.Context, client.Reader) error
		})
		if !ok {
			return nil
		}
		before := mg.DeepCopyObject().(client.Object)
		if err := rr.ResolveReferences(ctx, c); err != nil {
			return errors.Wrap(err, "cannot resolve references")
		}
		patch := client.MergeFromWithOptions(before, client.MergeFromWithOptimisticLock{})
		data, err := client.MergeFrom(before).Data(mg)
		if err != nil {
			return errors.Wrap(err, "cannot prepare reference patch")
		}
		if string(data) == "{}" {
			return nil
		}
		return errors.Wrap(c.Patch(ctx, mg, patch), "cannot persist resolved references")
	})
}
