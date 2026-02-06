package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		wantPort    int
		wantNASPath string
		wantErr     bool
	}{
		{
			name:        "Default values",
			envVars:     map[string]string{},
			wantPort:    3001,
			wantNASPath: "/nas",
			wantErr:     false,
		},
		{
			name: "Custom port",
			envVars: map[string]string{
				"PORT": "8080",
			},
			wantPort:    8080,
			wantNASPath: "/nas",
			wantErr:     false,
		},
		{
			name: "Custom NAS path",
			envVars: map[string]string{
				"NAS_PATH": "/mnt/storage",
			},
			wantPort:    3001,
			wantNASPath: "/mnt/storage",
			wantErr:     false,
		},
		{
			name: "Invalid port - too low",
			envVars: map[string]string{
				"PORT": "0",
			},
			wantErr: true,
		},
		{
			name: "Invalid port - too high",
			envVars: map[string]string{
				"PORT": "99999",
			},
			wantErr: true,
		},
		{
			name: "Production environment",
			envVars: map[string]string{
				"ENVIRONMENT": "production",
			},
			wantPort: 3001,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env vars
			originalEnv := make(map[string]string)
			envKeys := []string{"PORT", "DATA_DIR", "NAS_PATH", "LOG_LEVEL", "ENVIRONMENT"}
			for _, key := range envKeys {
				originalEnv[key] = os.Getenv(key)
				os.Unsetenv(key)
			}
			defer func() {
				// Restore original env vars
				for key, val := range originalEnv {
					if val != "" {
						os.Setenv(key, val)
					}
				}
			}()

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			cfg, err := Load()

			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if cfg.Port != tt.wantPort {
				t.Errorf("Load() Port = %v, want %v", cfg.Port, tt.wantPort)
			}

			if tt.wantNASPath != "" && cfg.NASPath != tt.wantNASPath {
				t.Errorf("Load() NASPath = %v, want %v", cfg.NASPath, tt.wantNASPath)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "Valid config",
			cfg: &Config{
				Port:     3001,
				DataDir:  "./data",
				LogLevel: "info",
			},
			wantErr: false,
		},
		{
			name: "Invalid port - too low",
			cfg: &Config{
				Port:     0,
				DataDir:  "./data",
				LogLevel: "info",
			},
			wantErr: true,
		},
		{
			name: "Invalid port - too high",
			cfg: &Config{
				Port:     99999,
				DataDir:  "./data",
				LogLevel: "info",
			},
			wantErr: true,
		},
		{
			name: "Empty data directory",
			cfg: &Config{
				Port:     3001,
				DataDir:  "",
				LogLevel: "info",
			},
			wantErr: true,
		},
		{
			name: "Invalid log level",
			cfg: &Config{
				Port:     3001,
				DataDir:  "./data",
				LogLevel: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsDevelopment(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		want        bool
	}{
		{
			name:        "Development environment",
			environment: "development",
			want:        true,
		},
		{
			name:        "Production environment",
			environment: "production",
			want:        false,
		},
		{
			name:        "Empty environment",
			environment: "",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Environment: tt.environment}
			if got := cfg.IsDevelopment(); got != tt.want {
				t.Errorf("IsDevelopment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsProduction(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		want        bool
	}{
		{
			name:        "Production environment",
			environment: "production",
			want:        true,
		},
		{
			name:        "Development environment",
			environment: "development",
			want:        false,
		},
		{
			name:        "Empty environment",
			environment: "",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Environment: tt.environment}
			if got := cfg.IsProduction(); got != tt.want {
				t.Errorf("IsProduction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnsureDirectories(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()

	cfg := &Config{
		DataDir:      tmpDir + "/data",
		CSVDir:       tmpDir + "/data/csv",
		DownloadsDir: tmpDir + "/data/downloads",
	}

	err := cfg.EnsureDirectories()
	if err != nil {
		t.Fatalf("EnsureDirectories() error = %v", err)
	}

	// Check if directories were created
	dirs := []string{cfg.DataDir, cfg.CSVDir, cfg.DownloadsDir}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", dir)
		}
	}
}
