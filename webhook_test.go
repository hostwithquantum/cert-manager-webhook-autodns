package main_test

import (
	"testing"

	cmwebhook "github.com/cert-manager/cert-manager/pkg/acme/webhook"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apiserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/rest"

	webhook "github.com/hostwithquantum/cert-manager-webhook-autodns"
)

// TestResourceName ensures this to be lowercase (k8s requires this now)
func TestResourceName(t *testing.T) {
	wh := &webhook.AutoDNSProviderSolver{}
	assert.Equal(t, "autodns", wh.Name())
}

// TestServerStartup does what cmd.RunWebhookServer does on boot.
// This is to catch errors (required changes) early (e.g. non-lowercase solver
// Name() before we release.
func TestServerStartup(t *testing.T) {
	cfg := genericapiserver.NewRecommendedConfig(apiserver.Codecs)
	cfg.ExternalAddress = "192.168.10.4:443"
	cfg.LoopbackClientConfig = &rest.Config{}

	_, err := (&apiserver.Config{
		GenericConfig: cfg,
		ExtraConfig: apiserver.ExtraConfig{
			SolverGroup: "acme.example.com",
			Solvers:     []cmwebhook.Solver{&webhook.AutoDNSProviderSolver{}},
		},
	}).Complete().New()
	require.NoError(t, err)
}
