// SPDX-License-Identifier: Apache-2.0

package references_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cluster "github.com/oxlev/provider-doppler/apis/cluster/secrets/v1alpha1"
	ns "github.com/oxlev/provider-doppler/apis/namespaced/secrets/v1alpha1"
	"github.com/oxlev/provider-doppler/internal/references"
)

type patchClient struct {
	client.Client
	patches int
	t       *testing.T
}

func (c *patchClient) Patch(ctx context.Context, obj client.Object, p client.Patch, opts ...client.PatchOption) error {
	c.t.Helper()
	if p.Type() != types.MergePatchType {
		c.t.Fatalf("reference delta must be a merge patch, got %s", p.Type())
	}
	data, err := p.Data(obj)
	if err != nil {
		c.t.Fatal(err)
	}
	var delta struct {
		Metadata struct {
			ResourceVersion string `json:"resourceVersion"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(data, &delta); err != nil {
		c.t.Fatal(err)
	}
	if delta.Metadata.ResourceVersion == "" {
		c.t.Fatal("reference patch must use optimistic locking")
	}
	c.patches++
	return c.Client.Patch(ctx, obj, p, opts...)
}

func TestIncrementalResolutionPreservesExistingFields(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	if err := ns.SchemeBuilder.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	project := &ns.Project{ObjectMeta: metav1.ObjectMeta{Name: "project", Namespace: "test"}}
	config := &ns.Config{ObjectMeta: metav1.ObjectMeta{Name: "config", Namespace: "test"}}
	for _, m := range []resource.Managed{project, config} {
		meta.SetExternalName(m, m.GetName())
		m.SetConditions(xpv2.Available(), xpv2.ReconcileSuccess())
	}
	secret := &ns.Secret{ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "test"}}
	secret.Spec.ForProvider.ProjectRef = &xpv2.NamespacedReference{Name: "project"}
	secret.Spec.ForProvider.Config = ptr.To("original")
	secret.Spec.ForProvider.ConfigRef = &xpv2.NamespacedReference{Name: "config"}
	c := &patchClient{Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(project, config, secret).Build(), t: t}
	if err := c.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
		t.Fatal(err)
	}
	resolver := references.NewResolver(c)
	if err := resolver.ResolveReferences(ctx, secret); err != nil {
		t.Fatal(err)
	}
	// A later reference change must not relinquish the project field resolved above.
	secret.Spec.ForProvider.Config = nil
	if err := c.Update(ctx, secret); err != nil {
		t.Fatal(err)
	}
	if err := resolver.ResolveReferences(ctx, secret); err != nil {
		t.Fatal(err)
	}
	if ptr.Deref(secret.Spec.ForProvider.Project, "") != "project" || ptr.Deref(secret.Spec.ForProvider.Config, "") != "config" {
		t.Fatal("resolved fields lost")
	}
	n := c.patches
	if err := resolver.ResolveReferences(ctx, secret); err != nil {
		t.Fatal(err)
	}
	if c.patches != n {
		t.Fatal("stable references should not be patched")
	}
}

func TestReadyExternalName(t *testing.T) {
	p := &ns.Project{}
	meta.SetExternalName(p, "desired-name")
	extract := references.ReadyExternalName()
	if extract(p) != "" {
		t.Fatal("uncreated parent was resolved")
	}
	p.SetConditions(xpv2.Available())
	if extract(p) != "" {
		t.Fatal("unsynced parent was resolved")
	}
	p.SetConditions(xpv2.ReconcileSuccess())
	if extract(p) != "desired-name" {
		t.Fatal("ready parent not resolved")
	}
}

func TestClusterReferenceWaitsForReadyParent(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	if err := cluster.SchemeBuilder.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	project := &cluster.Project{ObjectMeta: metav1.ObjectMeta{Name: "project"}}
	meta.SetExternalName(project, "project-slug")
	environment := &cluster.Environment{ObjectMeta: metav1.ObjectMeta{Name: "env"}}
	environment.Spec.ForProvider.ProjectRef = &xpv2.Reference{Name: "project"}
	c := &patchClient{Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(project, environment).Build(), t: t}
	if err := c.Get(ctx, client.ObjectKeyFromObject(environment), environment); err != nil {
		t.Fatal(err)
	}
	resolver := references.NewResolver(c)
	if err := resolver.ResolveReferences(ctx, environment); err == nil {
		t.Fatal("unready parent resolved")
	}
	if c.patches != 0 {
		t.Fatal("failed resolution must not be persisted")
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(project), project); err != nil {
		t.Fatal(err)
	}
	project.SetConditions(xpv2.Available(), xpv2.ReconcileSuccess())
	if err := c.Update(ctx, project); err != nil {
		t.Fatal(err)
	}
	if err := resolver.ResolveReferences(ctx, environment); err != nil {
		t.Fatal(err)
	}
	if ptr.Deref(environment.Spec.ForProvider.Project, "") != "project-slug" {
		t.Fatal("ready parent not resolved")
	}
}

var _ managed.ReferenceResolver = references.NewResolver(nil)
