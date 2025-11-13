package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	mdns "github.com/miekg/dns"

	"github.com/cert-manager/cert-manager/test/acme/dns"
)

var (
	zone = os.Getenv("TEST_ZONE_NAME")
	fqdn = os.Getenv("TEST_FQDN")
)

type recordStore struct {
	mu      sync.Mutex
	records map[string][]string
}

func (s *recordStore) add(name, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name = mdns.Fqdn(name)
	s.records[name] = append(s.records[name], value)
}

func (s *recordStore) remove(name, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name = mdns.Fqdn(name)
	kept := s.records[name][:0]
	for _, v := range s.records[name] {
		if v != value {
			kept = append(kept, v)
		}
	}
	if len(kept) == 0 {
		delete(s.records, name)
	} else {
		s.records[name] = kept
	}
}

func (s *recordStore) get(name string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.records[mdns.Fqdn(name)]...)
}

func startDNSServer(store *recordStore) (string, func() error, error) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	server := &mdns.Server{
		PacketConn: pc,
		Handler: mdns.HandlerFunc(func(w mdns.ResponseWriter, r *mdns.Msg) {
			m := new(mdns.Msg)
			m.SetReply(r)
			m.Authoritative = true
			for _, q := range r.Question {
				if q.Qtype != mdns.TypeTXT {
					continue
				}
				for _, v := range store.get(q.Name) {
					m.Answer = append(m.Answer, &mdns.TXT{
						Hdr: mdns.RR_Header{Name: q.Name, Rrtype: mdns.TypeTXT, Class: mdns.ClassINET, Ttl: 1},
						Txt: []string{v},
					})
				}
			}
			_ = w.WriteMsg(m)
		}),
	}
	started := make(chan struct{})
	server.NotifyStartedFunc = func() { close(started) }
	go func() {
		_ = server.ActivateAndServe()
		_ = pc.Close()
	}()
	<-started
	return pc.LocalAddr().String(), server.Shutdown, nil
}

func startAPIServer(t *testing.T, store *recordStore) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.Error(w, "unexpected method "+r.Method, http.StatusMethodNotAllowed)
			return
		}
		var body AutoDNSData
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request body: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for _, rec := range body.ResourceRecordsAdd {
			if rec.Type == "TXT" {
				store.add(rec.Name, rec.Value)
			}
		}
		for _, rec := range body.ResourceRecordsRem {
			if rec.Type == "TXT" {
				store.remove(rec.Name, rec.Value)
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
}

func TestRunsSuite(t *testing.T) {
	store := &recordStore{records: make(map[string][]string)}

	dnsAddr, dnsShutdown, err := startDNSServer(store)
	if err != nil {
		t.Fatalf("start dns server: %v", err)
	}
	defer func() { _ = dnsShutdown() }()

	apiServer := startAPIServer(t, store)
	defer apiServer.Close()

	configPath := filepath.Join("testdata", "autoDNS", "config.json")
	cfg := fmt.Sprintf(`{
    "zone": "example.com",
    "nameserver": "ns1.example.com",
    "context": "12345",
    "url": %q,
    "username": "test",
    "password": "test"
}`, apiServer.URL)
	if err := os.WriteFile(configPath, []byte(cfg), 0644); err != nil {
		t.Fatalf("write %s: %v", configPath, err)
	}

	fixture := dns.NewFixture(&autoDNSProviderSolver{},
		dns.SetResolvedZone(zone),
		dns.SetResolvedFQDN(fqdn),
		dns.SetAllowAmbientCredentials(false),
		dns.SetManifestPath("testdata/autoDNS"),
		dns.SetDNSServer(dnsAddr),
		dns.SetUseAuthoritative(false),
		dns.SetPollInterval(500*time.Millisecond),
		dns.SetPropagationLimit(10*time.Second),
	)

	fixture.RunConformance(t)
}
