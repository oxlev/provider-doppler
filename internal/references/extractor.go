// SPDX-License-Identifier: Apache-2.0

package references

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reference"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
)

// ReadyExternalName prevents a desired name from being used before the parent
// actually exists. Environments and configs have external names before creation.
func ReadyExternalName() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		if mg.GetCondition(xpv2.TypeReady).Status != corev1.ConditionTrue ||
			mg.GetCondition(xpv2.TypeSynced).Status != corev1.ConditionTrue {
			return ""
		}
		return meta.GetExternalName(mg)
	}
}
