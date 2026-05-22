package main_test

import (
	"testing"

	webhook "github.com/hostwithquantum/cert-manager-webhook-autodns"
	"github.com/stretchr/testify/assert"
)

// TestResourceName ensures this to be lowercase (k8s requires this now)
func TestResourceName(t *testing.T) {
	wh := &webhook.AutoDNSProviderSolver{}
	assert.Equal(t, "autodns", wh.Name())
}
