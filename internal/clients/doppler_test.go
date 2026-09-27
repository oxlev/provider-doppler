package clients

import (
	"context"
	"strings"
	"testing"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	secrets "github.com/oxlev/provider-doppler/apis/namespaced/secrets/v1alpha1"
	pc "github.com/oxlev/provider-doppler/apis/namespaced/v1beta1"
)

func TestNamespacedCredentialIsolation(t *testing.T) {
	scheme := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{corev1.AddToScheme, secrets.SchemeBuilder.AddToScheme, pc.SchemeBuilder.AddToScheme} {
		if err := add(scheme); err != nil {
			t.Fatal(err)
		}
	}
	config := &pc.ProviderConfig{ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "tenant"}}
	config.Spec.Credentials.Source = xpv2.CredentialsSourceSecret
	config.Spec.Credentials.SecretRef = &xpv2.SecretKeySelector{
		SecretReference: xpv2.SecretReference{Name: "creds", Namespace: "other"}, Key: "credentials",
	}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "creds", Namespace: "tenant"}, Data: map[string][]byte{"credentials": []byte(`{"doppler_token":"tenant-token"}`)}}
	other := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "creds", Namespace: "other"}, Data: map[string][]byte{"credentials": []byte(`{"doppler_token":"other-token"}`)}}
	mg := &secrets.Project{ObjectMeta: metav1.ObjectMeta{Name: "example", Namespace: "tenant", UID: "test-uid"}}
	mg.SetProviderConfigReference(&xpv2.ProviderConfigReference{Name: "default", Kind: "ProviderConfig"})
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(config, secret, other).Build()
	setup, err := TerraformSetupBuilder("1.5.7", "dopplerhq/doppler", "1.21.5")(context.Background(), c, mg)
	if err != nil {
		t.Fatal(err)
	}
	if setup.Configuration["doppler_token"] != "tenant-token" {
		t.Fatal("credential namespace boundary violated")
	}
	usages := &pc.ProviderConfigUsageList{}
	if err := c.List(context.Background(), usages); err != nil {
		t.Fatal(err)
	}
	if len(usages.Items) != 1 {
		t.Fatalf("want one ProviderConfigUsage, got %d", len(usages.Items))
	}
}

func TestCredentialConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name, data string
		valid      bool
	}{
		{"valid", `{"doppler_token":"test-token"}`, true},
		{"missing", `{}`, false},
		{"null", `null`, false},
		{"blank", `{"doppler_token":"  "}`, false},
		{"wrong type", `{"doppler_token":123}`, false},
		{"invalid JSON", `{"doppler_token":"test-token"`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config, err := credentialConfiguration([]byte(tc.data))
			if tc.valid {
				if err != nil || config["doppler_token"] != "test-token" {
					t.Fatalf("valid credential rejected")
				}
			} else {
				if err == nil {
					t.Fatal("invalid credential accepted")
				}
				if strings.Contains(err.Error(), "test-token") {
					t.Fatal("credential leaked in error")
				}
			}
		})
	}
}
