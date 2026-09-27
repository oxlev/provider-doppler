package config

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/crossplane/upjet/v2/pkg/config"
)

// ExternalNameConfigs explicitly allowlists resources with reviewed import IDs.
// Projects use a server-generated slug; their display name remains in forProvider.
var ExternalNameConfigs = map[string]config.ExternalName{
	"doppler_project":     config.IdentifierFromProvider,
	"doppler_environment": config.TemplatedStringAsIdentifier("slug", "{{ .parameters.project }}.{{ .external_name }}"),
	"doppler_config":      configExternalName(),
	"doppler_secret":      config.TemplatedStringAsIdentifier("name", "{{ .parameters.project }}.{{ .parameters.config }}.{{ .external_name }}"),
}

// configExternalName keeps desired naming separate from confirmed identity.
// Doppler returns an ambiguous authorization error when reading a missing config;
// fabricating an ID before creation would make Terraform refresh fail forever.
func configExternalName() config.ExternalName {
	base := config.TemplatedStringAsIdentifier("name", "{{ .parameters.project }}.{{ .parameters.environment }}.{{ .external_name }}")
	e := config.NewExternalNameFrom(base,
		config.WithSetIdentifierArgumentsFn(func(_ config.SetIdentifierArgumentsFn, params map[string]any, externalName string) {
			// Preserve an explicit name so GetIDFn can reject identity changes.
			if name, _ := params["name"].(string); name == "" && externalName != "" {
				params["name"] = externalName
			}
		}),
		config.WithGetIDFn(func(fn config.GetIDFn, ctx context.Context, externalName string, params, setup map[string]any) (string, error) {
			name, _ := params["name"].(string)
			if externalName == "" {
				if strings.TrimSpace(name) == "" {
					return "", fmt.Errorf("new configs require spec.forProvider.name (or initProvider.name); use external-name only to adopt an existing config")
				}
				return "", nil
			}
			if name != "" && name != externalName {
				return "", fmt.Errorf("config name must match the confirmed external-name; renaming is not supported")
			}
			return fn(ctx, externalName, params, setup)
		}),
	)
	e.OmittedFields = nil
	e.DisableNameInitializer = true
	return e
}

// ExternalNameConfigurations applies the reviewed external-name mappings.
func ExternalNameConfigurations() config.ResourceOption {
	return func(r *config.Resource) {
		if e, ok := ExternalNameConfigs[r.Name]; ok {
			r.ExternalName = e
		}
	}
}

// ExternalNameConfigured returns deterministic, exact-match resource patterns.
func ExternalNameConfigured() []string {
	names := make([]string, 0, len(ExternalNameConfigs))
	for name := range ExternalNameConfigs {
		names = append(names, "^"+name+"$")
	}
	sort.Strings(names)
	return names
}
