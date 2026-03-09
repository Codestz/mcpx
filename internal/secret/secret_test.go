package secret

import (
	"fmt"
	"testing"
)

// MemoryStore is an in-memory Store implementation for testing.
type MemoryStore struct {
	data map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[string]string)}
}

func (m *MemoryStore) Get(name string) (string, error) {
	v, ok := m.data[name]
	if !ok {
		return "", fmt.Errorf("secret %q not found", name)
	}
	return v, nil
}

func (m *MemoryStore) Set(name, value string) error {
	m.data[name] = value
	return nil
}

func (m *MemoryStore) Delete(name string) error {
	if _, ok := m.data[name]; !ok {
		return fmt.Errorf("secret %q not found", name)
	}
	delete(m.data, name)
	return nil
}

func (m *MemoryStore) List() ([]string, error) {
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys, nil
}

func TestGet(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		seed    map[string]string
		want    string
		wantErr bool
	}{
		{
			name: "found",
			key:  "api_key",
			seed: map[string]string{"api_key": "abc123"},
			want: "abc123",
		},
		{
			name:    "not found",
			key:     "missing",
			seed:    map[string]string{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMemoryStore()
			for k, v := range tt.seed {
				s.data[k] = v
			}
			got, err := s.Get(tt.key)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Get() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Get() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSet(t *testing.T) {
	s := NewMemoryStore()

	if err := s.Set("token", "xyz"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, err := s.Get("token")
	if err != nil {
		t.Fatalf("Get() after Set() error = %v", err)
	}
	if got != "xyz" {
		t.Errorf("Get() = %q, want %q", got, "xyz")
	}

	// Overwrite existing key.
	if err := s.Set("token", "new"); err != nil {
		t.Fatalf("Set() overwrite error = %v", err)
	}
	got, err = s.Get("token")
	if err != nil {
		t.Fatalf("Get() after overwrite error = %v", err)
	}
	if got != "new" {
		t.Errorf("Get() = %q, want %q", got, "new")
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		seed    map[string]string
		wantErr bool
	}{
		{
			name: "delete existing",
			key:  "token",
			seed: map[string]string{"token": "abc"},
		},
		{
			name:    "delete missing",
			key:     "nope",
			seed:    map[string]string{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMemoryStore()
			for k, v := range tt.seed {
				s.data[k] = v
			}
			err := s.Delete(tt.key)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if _, err := s.Get(tt.key); err == nil {
					t.Error("Get() after Delete() should fail")
				}
			}
		})
	}
}

func TestList(t *testing.T) {
	s := NewMemoryStore()

	// Empty store.
	keys, err := s.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("List() = %v, want empty", keys)
	}

	// After adding keys.
	s.Set("a", "1")
	s.Set("b", "2")
	keys, err = s.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("List() len = %d, want 2", len(keys))
	}

	found := make(map[string]bool)
	for _, k := range keys {
		found[k] = true
	}
	if !found["a"] || !found["b"] {
		t.Errorf("List() = %v, want [a, b]", keys)
	}
}
