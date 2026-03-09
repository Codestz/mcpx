package cli

import "testing"

func TestValidateSecretName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "api_key", false},
		{"valid with dots", "github.token", false},
		{"valid with dash", "my-secret", false},
		{"valid mixed", "API_Key.v2", false},
		{"empty", "", true},
		{"starts with dot", ".hidden", true},
		{"starts with dash", "-flag", true},
		{"has space", "my secret", true},
		{"has special char", "key@host", true},
		{"has slash", "path/key", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSecretName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSecretName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
