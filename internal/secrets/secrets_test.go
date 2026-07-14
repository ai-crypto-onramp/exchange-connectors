package secrets

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeVault struct {
	mu          sync.Mutex
	secrets     map[string]*Secret
	revoked     []string
	rotateCount int
}

func newFakeVault() *fakeVault {
	return &fakeVault{
		secrets: map[string]*Secret{
			"secret/exchange-connectors/binance/api-key":    {Data: map[string]string{"value": "key-v1"}, Version: "v1", LeaseID: "lease-1"},
			"secret/exchange-connectors/binance/api-secret": {Data: map[string]string{"value": "secret-v1"}, Version: "v1", LeaseID: "lease-2"},
		},
	}
}

func (f *fakeVault) Read(ctx context.Context, path string) (*Secret, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.secrets[path]
	if !ok {
		return nil, ErrSecretNotFound
	}
	cp := *s
	return &cp, nil
}

func (f *fakeVault) Revoke(ctx context.Context, leaseID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.revoked = append(f.revoked, leaseID)
	return nil
}

func (f *fakeVault) rotate() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rotateCount++
	v := "v" + strings.Repeat("2", f.rotateCount)
	f.secrets["secret/exchange-connectors/binance/api-key"] = &Secret{Data: map[string]string{"value": "key-" + v}, Version: v, LeaseID: "lease-1-" + v}
	f.secrets["secret/exchange-connectors/binance/api-secret"] = &Secret{Data: map[string]string{"value": "secret-" + v}, Version: v, LeaseID: "lease-2-" + v}
}

func TestLoad(t *testing.T) {
	v := newFakeVault()
	m := NewManager("binance", v)
	creds, err := m.Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if creds.APIKey != "key-v1" {
		t.Fatalf("key: %s", creds.APIKey)
	}
	if creds.APISecret != "secret-v1" {
		t.Fatalf("secret: %s", creds.APISecret)
	}
	if creds.Version != "v1" {
		t.Fatalf("version: %s", creds.Version)
	}
	if m.Current().APIKey != "key-v1" {
		t.Fatalf("current: %s", m.Current().APIKey)
	}
}

func TestLoadMissing(t *testing.T) {
	v := newFakeVault()
	m := NewManager("kraken", v)
	_, err := m.Load(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestRotateWithProbe(t *testing.T) {
	v := newFakeVault()
	m := NewManager("binance", v)
	_, _ = m.Load(context.Background())
	oldLease := m.Current().LeaseID
	v.rotate()
	probed := false
	creds, err := m.Rotate(context.Background(), func(ctx context.Context, c *Credentials) error {
		probed = true
		if c.APIKey != "key-v2" {
			t.Fatalf("probe key: %s", c.APIKey)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if !probed {
		t.Fatalf("probe not called")
	}
	if creds.APIKey != "key-v2" {
		t.Fatalf("new key: %s", creds.APIKey)
	}
	revoked := false
	for _, r := range v.revoked {
		if r == oldLease {
			revoked = true
		}
	}
	if !revoked {
		t.Fatalf("old lease not revoked")
	}
}

func TestRotateProbeFailsNoRevoke(t *testing.T) {
	v := newFakeVault()
	m := NewManager("binance", v)
	_, _ = m.Load(context.Background())
	v.rotate()
	_, err := m.Rotate(context.Background(), func(ctx context.Context, c *Credentials) error {
		return context.DeadlineExceeded
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if len(v.revoked) != 0 {
		t.Fatalf("should not revoke on probe failure")
	}
}

func TestRevokeOld(t *testing.T) {
	v := newFakeVault()
	m := NewManager("binance", v)
	_, _ = m.Load(context.Background())
	if err := m.RevokeOld(context.Background()); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if len(v.revoked) != 1 {
		t.Fatalf("revoked: %d", len(v.revoked))
	}
}

func TestRedactShort(t *testing.T) {
	if Redact("abc") != "***" {
		t.Fatalf("redact short: %s", Redact("abc"))
	}
}

func TestRedactLong(t *testing.T) {
	r := Redact("abcdefghij")
	if r != "ab******ij" {
		t.Fatalf("redact: %s", r)
	}
}

func TestRedactEmpty(t *testing.T) {
	if Redact("") != "" {
		t.Fatalf("redact empty: %s", Redact(""))
	}
}

func TestRedactCreds(t *testing.T) {
	c := &Credentials{APIKey: "abcdefghij", APISecret: "1234567890"}
	r := RedactCreds(c)
	if r.APIKey == c.APIKey {
		t.Fatalf("key not redacted: %s", r.APIKey)
	}
	if r.APISecret == c.APISecret {
		t.Fatalf("secret not redacted: %s", r.APISecret)
	}
}

func TestRedactCredsNil(t *testing.T) {
	if RedactCreds(nil) != nil {
		t.Fatalf("expected nil")
	}
}

func TestSecretNoDiskPersistence(t *testing.T) {
	v := newFakeVault()
	m := NewManager("binance", v)
	creds, _ := m.Load(context.Background())
	_ = creds
	if time.Since(m.Current().LoadedAt) > 5*time.Second {
		t.Fatalf("loaded time too old")
	}
}

func TestCredentialStoreRecord(t *testing.T) {
	s := NewCredentialStore()
	s.Record("binance", "ab**ij", "secret/exchange-connectors/binance/api-key", "v1", time.Now())
	if len(s.Rows()) != 1 {
		t.Fatalf("rows: %d", len(s.Rows()))
	}
	r := s.Rows()[0]
	if r.Venue != "binance" || r.Version != "v1" {
		t.Fatalf("row: %+v", r)
	}
	if r.KeyID == "abcdefghij" {
		t.Fatalf("keyID should be redacted, got %s", r.KeyID)
	}
}

func TestCredentialStoreLatest(t *testing.T) {
	s := NewCredentialStore()
	s.Record("binance", "k1", "p1", "v1", time.Now())
	s.Record("binance", "k2", "p2", "v2", time.Now())
	s.Record("kraken", "k3", "p3", "v3", time.Now())
	latest := s.Latest("binance")
	if latest == nil || latest.Version != "v2" {
		t.Fatalf("latest: %+v", latest)
	}
	if s.Latest("unknown") != nil {
		t.Fatalf("expected nil")
	}
}

func TestRecordCredentials(t *testing.T) {
	s := NewCredentialStore()
	c := &Credentials{APIKey: "abcdefghij", APISecret: "secret", Version: "v1", LoadedAt: time.Now()}
	RecordCredentials(s, "binance", c)
	r := s.Latest("binance")
	if r == nil || r.Version != "v1" {
		t.Fatalf("record: %+v", r)
	}
	if r.KeyID == "abcdefghij" {
		t.Fatalf("keyID not redacted")
	}
}