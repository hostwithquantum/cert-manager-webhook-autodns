package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/rest"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"

	"github.com/libdns/autodns/sdk"
)

var GroupName = os.Getenv("GROUP_NAME")

func main() {
	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}

	cmd.RunWebhookServer(GroupName,
		&AutoDNSProviderSolver{},
	)
}

type AutoDNSProviderSolver struct{}

type solverConfig struct {
	Zone       string `json:"zone"`
	NameServer string `json:"nameserver"`
	Context    string `json:"context"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	URL        string `json:"url"`
}

func (c *AutoDNSProviderSolver) Name() string {
	return "autoDNS"
}

func (c *AutoDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return err
	}

	if cfg.Zone == "" {
		cfg.Zone = ch.ResolvedZone
	}

	client := &sdk.SDK{
		Username: cfg.Username,
		Password: cfg.Password,
		Context:  cfg.Context,
		Endpoint: cfg.URL,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = client.PatchZone(ctx, cfg.Zone, cfg.NameServer, sdk.ZonePatch{
		ResourceRecordsAdd: []sdk.ZoneRecord{
			{
				Name:  ch.ResolvedFQDN,
				Type:  "TXT",
				TTL:   60,
				Value: ch.Key,
			},
		},
	})
	return err
}

func (c *AutoDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return err
	}

	if cfg.Zone == "" {
		cfg.Zone = ch.ResolvedZone
	}

	client := &sdk.SDK{
		Username: cfg.Username,
		Password: cfg.Password,
		Context:  cfg.Context,
		Endpoint: cfg.URL,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = client.PatchZone(ctx, cfg.Zone, cfg.NameServer, sdk.ZonePatch{
		ResourceRecordsRem: []sdk.ZoneRecord{
			{
				Name:  ch.ResolvedFQDN,
				Type:  "TXT",
				TTL:   60,
				Value: ch.Key,
			},
		},
	})
	return err
}

func (c *AutoDNSProviderSolver) Initialize(_ *rest.Config, _ <-chan struct{}) error {
	return nil
}

func loadConfig(cfgJSON *extapi.JSON) (*solverConfig, error) {
	cfg := solverConfig{}
	if cfgJSON == nil {
		return nil, fmt.Errorf("missing config")
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return nil, fmt.Errorf("error decoding solver config: %v", err)
	}

	return &cfg, nil
}
