package config

import (
	"context"
	"testing"
)

func TestConfigIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, external, desired, wantID string
		wantErr                         bool
	}{
		{name: "create", desired: "ci_build"},
		{name: "missing name", wantErr: true},
		{name: "blank name", desired: "  ", wantErr: true},
		{name: "legacy adoption", external: "ci_build", wantID: "app.ci.ci_build"},
		{name: "confirmed creation", external: "ci_build", desired: "ci_build", wantID: "app.ci.ci_build"},
		{name: "reject rename", external: "ci_build", desired: "ci_other", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := ExternalNameConfigs["doppler_config"]
			p := map[string]any{"project": "app", "environment": "ci"}
			if tc.desired != "" {
				p["name"] = tc.desired
			}
			e.SetIdentifierArgumentFn(p, tc.external)
			id, err := e.GetIDFn(context.Background(), tc.external, p, nil)
			if (err != nil) != tc.wantErr || id != tc.wantID {
				t.Fatalf("id=%q err=%v", id, err)
			}
			if !e.DisableNameInitializer || len(e.OmittedFields) != 0 {
				t.Fatal("desired name must remain in spec, without name initialization")
			}
		})
	}
	for _, p := range []string{"cluster", "namespaced"} {
		r := provider(p, "doppler.crossplane.io").Resources["doppler_config"]
		if r.TerraformResource.Schema["name"].Required || !r.TerraformResource.Schema["name"].Optional {
			t.Fatal("legacy configs without spec name must remain valid")
		}
	}
}
