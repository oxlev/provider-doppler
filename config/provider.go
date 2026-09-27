package config

import (
	_ "embed"
	"fmt"

	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
)

const modulePath = "github.com/oxlev/provider-doppler"

//go:embed schema.json
var providerSchema string

//go:embed provider-metadata.yaml
var providerMetadata string

//go:embed templates/controller.go.tmpl
var controllerTemplate string

// GetProvider returns the legacy cluster-scoped configuration.
func GetProvider() *ujconfig.Provider { return provider("cluster", "doppler.crossplane.io") }

// GetProviderNamespaced returns the Crossplane v2 namespaced configuration.
func GetProviderNamespaced() *ujconfig.Provider {
	return provider("namespaced", "doppler.m.crossplane.io")
}

func provider(scope, group string) *ujconfig.Provider {
	p := ujconfig.NewProvider([]byte(providerSchema), "doppler", modulePath, []byte(providerMetadata),
		ujconfig.WithRootGroup(group),
		ujconfig.WithIncludeList(ExternalNameConfigured()),
		ujconfig.WithFeaturesPackage("internal/features"),
		ujconfig.WithControllerTemplate(controllerTemplate),
		ujconfig.WithDefaultResourceOptions(ExternalNameConfigurations()),
	)
	for _, name := range []string{"doppler_project", "doppler_environment", "doppler_config", "doppler_secret"} {
		p.AddResourceConfigurator(name, func(r *ujconfig.Resource) {
			r.ShortGroup = "secrets"
			r.Version = "v1alpha1"
			if r.Name != "doppler_project" {
				r.References["project"] = ujconfig.Reference{Type: fmt.Sprintf("%s/apis/%s/secrets/v1alpha1.Project", modulePath, scope), Extractor: modulePath + "/internal/references.ReadyExternalName()"}
			}
			if r.Name == "doppler_config" {
				// Existing annotated configs remain valid without a spec name.
				// The external-name adapter validates new configs at runtime.
				r.TerraformResource.Schema["name"].Required = false
				r.TerraformResource.Schema["name"].Optional = true
				r.References["environment"] = ujconfig.Reference{Type: fmt.Sprintf("%s/apis/%s/secrets/v1alpha1.Environment", modulePath, scope), Extractor: modulePath + "/internal/references.ReadyExternalName()"}
			}
			if r.Name == "doppler_secret" {
				r.References["config"] = ujconfig.Reference{Type: fmt.Sprintf("%s/apis/%s/secrets/v1alpha1.Config", modulePath, scope), Extractor: modulePath + "/internal/references.ReadyExternalName()"}
			}
		})
	}
	p.ConfigureResources()
	return p
}
