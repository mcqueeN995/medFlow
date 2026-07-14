package password

import (
	"strings"
	"testing"
)

func TestHash(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  false,
		},
		{
			name:     "long password",
			password: strings.Repeat("a", 1000),
			wantErr:  false,
		},
		{
			name:     "special characters",
			password: "p@$$w0rd!#$%^&*()",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := Hash(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("Hash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Проверяем формат хеша
				if !strings.HasPrefix(hash, "$argon2id$v=19$") {
					t.Errorf("Hash() invalid format: %s", hash)
				}

				// Проверяем, что хеши разные (соль случайная)
				hash2, _ := Hash(tt.password)
				if hash == hash2 {
					t.Error("Hash() generated same hash for same password (salt not random)")
				}
			}
		})
	}
}

func TestCompare(t *testing.T) {
	// Сначала создаем хеш
	password := "testPassword123"
	hash, err := Hash(password)
	if err != nil {
		t.Fatalf("Failed to create hash: %v", err)
	}

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
		wantErr  bool
	}{
		{
			name:     "correct password",
			password: password,
			hash:     hash,
			want:     true,
			wantErr:  false,
		},
		{
			name:     "wrong password",
			password: "wrongPassword",
			hash:     hash,
			want:     false,
			wantErr:  false,
		},
		{
			name:     "invalid hash format",
			password: password,
			hash:     "invalid_hash",
			want:     false,
			wantErr:  true,
		},
		{
			name:     "empty password correct",
			password: "",
			hash:     func() string { h, _ := Hash(""); return h }(),
			want:     true,
			wantErr:  false,
		},
		{
			name:     "empty password wrong",
			password: "notempty",
			hash:     func() string { h, _ := Hash(""); return h }(),
			want:     false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Compare(tt.password, tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compare() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Compare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompare_InvalidFormats(t *testing.T) {
	tests := []struct {
		name    string
		hash    string
		wantErr bool
	}{
		{
			name:    "too few parts",
			hash:    "$argon2id$v=19$salt",
			wantErr: true,
		},
		{
			name:    "wrong algorithm",
			hash:    "$argon2i$v=19$m=65536,t=3,p=4$c2FsdA$aGFzaA",
			wantErr: true,
		},
		{
			name:    "wrong version",
			hash:    "$argon2id$v=18$m=65536,t=3,p=4$c2FsdA$aGFzaA",
			wantErr: true,
		},
		{
			name:    "invalid parameters",
			hash:    "$argon2id$v=19$invalid$m=65536,t=3,p=4$c2FsdA$aGFzaA",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Compare("password", tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compare() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func BenchmarkHash(b *testing.B) {
	password := "benchmarkPassword123"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Hash(password)
	}
}

func BenchmarkCompare(b *testing.B) {
	password := "benchmarkPassword123"
	hash, _ := Hash(password)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Compare(password, hash)
	}
}
