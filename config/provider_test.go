package config

import (
	"context"
	"reflect"
	"testing"
)

func TestExternalNames(t *testing.T) {
	for _, tc := range []struct {
		resource, name, id string
		parameters         map[string]any
	}{
		{"doppler_project", "app", "app", nil},
		{"doppler_environment", "dev", "app.dev", map[string]any{"project": "app"}},
		{"doppler_config", "dev_ci", "app.dev.dev_ci", map[string]any{"project": "app", "environment": "dev"}},
		{"doppler_secret", "API_KEY", "app.dev_ci.API_KEY", map[string]any{"project": "app", "config": "dev_ci"}},
	} {
		t.Run(tc.resource, func(t *testing.T) {
			e := ExternalNameConfigs[tc.resource]
			id, err := e.GetIDFn(context.Background(), tc.name, tc.parameters, nil)
			if err != nil || id != tc.id {
				t.Fatalf("ID = %q, %v; want %q", id, err, tc.id)
			}
			state := map[string]any{"id": tc.id, "name": tc.name, "slug": tc.name}
			name, err := e.GetExternalNameFn(state)
			if err != nil || name != tc.name {
				t.Fatalf("external name = %q, %v", name, err)
			}
		})
	}
}

func TestProviderConfiguration(t *testing.T) {
	for _, p := range []*struct{ scope string }{{"cluster"}, {"namespaced"}} {
		t.Run(p.scope, func(t *testing.T) {
			provider := provider(p.scope, "doppler.crossplane.io")
			if len(provider.Resources) != 4 {
				t.Fatalf("unexpected resources: %d", len(provider.Resources))
			}
			for name, r := range provider.Resources {
				if r.Version != "v1alpha1" {
					t.Errorf("%s must remain alpha until live acceptance tests", name)
				}
				if name != "doppler_project" && r.References["project"].Type == "" {
					t.Errorf("%s missing project reference", name)
				}
			}
			secret := provider.Resources["doppler_secret"].TerraformResource.Schema
			if !secret["value"].Sensitive || !secret["computed"].Sensitive {
				t.Fatal("secret fields must remain sensitive")
			}
		})
	}
	if !reflect.DeepEqual(ExternalNameConfigured(), ExternalNameConfigured()) {
		t.Fatal("non-deterministic allowlist")
	}
}
