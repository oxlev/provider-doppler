package config

import (
	"github.com/crossplane/upjet/v2/pkg/config"
	"sort"
)

// ExternalNameConfigs explicitly allowlists resources with reviewed import IDs.
// Projects use a server-generated slug; their display name remains in forProvider.
var ExternalNameConfigs = map[string]config.ExternalName{
	"doppler_project":     config.IdentifierFromProvider,
	"doppler_environment": config.TemplatedStringAsIdentifier("slug", "{{ .parameters.project }}.{{ .external_name }}"),
	"doppler_config":      config.TemplatedStringAsIdentifier("name", "{{ .parameters.project }}.{{ .parameters.environment }}.{{ .external_name }}"),
	"doppler_secret":      config.TemplatedStringAsIdentifier("name", "{{ .parameters.project }}.{{ .parameters.config }}.{{ .external_name }}"),
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
