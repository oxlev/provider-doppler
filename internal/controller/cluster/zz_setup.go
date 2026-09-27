// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/upjet/v2/pkg/controller"

	providerconfig "github.com/oxlev/provider-doppler/internal/controller/cluster/providerconfig"
	config "github.com/oxlev/provider-doppler/internal/controller/cluster/secrets/config"
	environment "github.com/oxlev/provider-doppler/internal/controller/cluster/secrets/environment"
	project "github.com/oxlev/provider-doppler/internal/controller/cluster/secrets/project"
	secret "github.com/oxlev/provider-doppler/internal/controller/cluster/secrets/secret"
)

// Setup creates all controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		providerconfig.Setup,
		config.Setup,
		environment.Setup,
		project.Setup,
		secret.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}

// SetupGated creates all controllers with the supplied logger and adds them to
// the supplied manager gated.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		providerconfig.SetupGated,
		config.SetupGated,
		environment.SetupGated,
		project.SetupGated,
		secret.SetupGated,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}

// SetupWebhookWithManager registers conversion webhooks for all resource kinds in the group.
func SetupWebhookWithManager(mgr ctrl.Manager) error {
	for _, setup := range []func(ctrl.Manager) error{
		providerconfig.SetupWebhookWithManager,
		config.SetupWebhookWithManager,
		environment.SetupWebhookWithManager,
		project.SetupWebhookWithManager,
		secret.SetupWebhookWithManager,
	} {
		if err := setup(mgr); err != nil {
			return err
		}
	}
	return nil
}
