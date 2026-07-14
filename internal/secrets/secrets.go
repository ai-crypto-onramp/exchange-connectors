package secrets

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var ErrSecretNotFound = errors.New("secrets: secret not found")

type Credentials struct {
	APIKey    string
	APISecret string
	Version   string
	LeaseID   string
	LoadedAt  time.Time
}

type VaultClient interface {
	Read(ctx context.Context, path string) (*Secret, error)
	Revoke(ctx context.Context, leaseID string) error
}

type Secret struct {
	Data      map[string]string
	Version   string
	LeaseID   string
	Renewable bool
}

type Manager struct {
	mu      sync.RWMutex
	venue   string
	client  VaultClient
	current *Credentials
}

func NewManager(venue string, client VaultClient) *Manager {
	return &Manager{venue: venue, client: client}
}

func (m *Manager) Load(ctx context.Context) (*Credentials, error) {
	keyPath := fmt.Sprintf("secret/exchange-connectors/%s/api-key", m.venue)
	secretPath := fmt.Sprintf("secret/exchange-connectors/%s/api-secret", m.venue)
	keySecret, err := m.client.Read(ctx, keyPath)
	if err != nil {
		return nil, fmt.Errorf("load api-key: %w", err)
	}
	secretSecret, err := m.client.Read(ctx, secretPath)
	if err != nil {
		return nil, fmt.Errorf("load api-secret: %w", err)
	}
	creds := &Credentials{
		APIKey:    keySecret.Data["value"],
		APISecret: secretSecret.Data["value"],
		Version:   keySecret.Version,
		LeaseID:   keySecret.LeaseID,
		LoadedAt:  time.Now().UTC(),
	}
	m.mu.Lock()
	m.current = creds
	m.mu.Unlock()
	return creds, nil
}

func (m *Manager) Current() *Credentials {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

func (m *Manager) Rotate(ctx context.Context, probe func(context.Context, *Credentials) error) (*Credentials, error) {
	old := m.Current()
	newCreds, err := m.Load(ctx)
	if err != nil {
		return nil, err
	}
	if probe != nil {
		if err := probe(ctx, newCreds); err != nil {
			return nil, fmt.Errorf("probe failed: %w", err)
		}
	}
	if old != nil && old.LeaseID != "" && old.LeaseID != newCreds.LeaseID {
		_ = m.client.Revoke(ctx, old.LeaseID)
	}
	return newCreds, nil
}

func (m *Manager) RevokeOld(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil || m.current.LeaseID == "" {
		return nil
	}
	return m.client.Revoke(ctx, m.current.LeaseID)
}

func Redact(s string) string {
	if len(s) == 0 {
		return ""
	}
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
}

func RedactCreds(c *Credentials) *Credentials {
	if c == nil {
		return nil
	}
	return &Credentials{
		APIKey:    Redact(c.APIKey),
		APISecret: Redact(c.APISecret),
		Version:   c.Version,
		LeaseID:   c.LeaseID,
		LoadedAt:  c.LoadedAt,
	}
}

type CredentialRecord struct {
	Venue     string    `json:"venue"`
	KeyID     string    `json:"key_id"`
	VaultPath string    `json:"vault_path"`
	Version   string    `json:"version"`
	RotatedAt time.Time `json:"rotated_at"`
}

type CredentialStore struct {
	mu    sync.Mutex
	rows  []CredentialRecord
}

func NewCredentialStore() *CredentialStore {
	return &CredentialStore{}
}

func (s *CredentialStore) Record(venue, keyID, vaultPath, version string, rotatedAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rows = append(s.rows, CredentialRecord{
		Venue:     venue,
		KeyID:     keyID,
		VaultPath: vaultPath,
		Version:   version,
		RotatedAt: rotatedAt,
	})
}

func (s *CredentialStore) Rows() []CredentialRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]CredentialRecord, len(s.rows))
	copy(out, s.rows)
	return out
}

func (s *CredentialStore) Latest(venue string) *CredentialRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.rows) - 1; i >= 0; i-- {
		if s.rows[i].Venue == venue {
			r := s.rows[i]
			return &r
		}
	}
	return nil
}

func RecordCredentials(store *CredentialStore, venue string, c *Credentials) {
	if store == nil || c == nil {
		return
	}
	store.Record(venue, Redact(c.APIKey), "secret/exchange-connectors/"+venue+"/api-key", c.Version, c.LoadedAt)
}