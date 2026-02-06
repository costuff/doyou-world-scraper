package logger

import (
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{
			name:  "Debug level",
			level: "debug",
		},
		{
			name:  "Info level",
			level: "info",
		},
		{
			name:  "Warn level",
			level: "warn",
		},
		{
			name:  "Error level",
			level: "error",
		},
		{
			name:  "Invalid level defaults to info",
			level: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(tt.level)
			if l == nil {
				t.Fatal("New() returned nil")
			}
		})
	}
}

func TestNewProduction(t *testing.T) {
	l := NewProduction("info")
	if l == nil {
		t.Fatal("NewProduction() returned nil")
	}
}

func TestInit(t *testing.T) {
	tests := []struct {
		name  string
		level string
		prod  bool
	}{
		{
			name:  "Development mode",
			level: "info",
			prod:  false,
		},
		{
			name:  "Production mode",
			level: "info",
			prod:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Init(tt.level, tt.prod)
			// Test that logging doesn't panic
			Info("test")
		})
	}
}

func TestLogging(t *testing.T) {
	Init("debug", false)

	t.Run("Debug", func(t *testing.T) {
		Debug("test debug")
	})

	t.Run("Debugf", func(t *testing.T) {
		Debugf("test %s", "debug")
	})

	t.Run("Info", func(t *testing.T) {
		Info("test info")
	})

	t.Run("Infof", func(t *testing.T) {
		Infof("test %s", "info")
	})

	t.Run("Warn", func(t *testing.T) {
		Warn("test warn")
	})

	t.Run("Warnf", func(t *testing.T) {
		Warnf("test %s", "warn")
	})

	t.Run("Error", func(t *testing.T) {
		Error("test error")
	})

	t.Run("Errorf", func(t *testing.T) {
		Errorf("test %s", "error")
	})
}

func TestLoggerMethods(t *testing.T) {
	l := New("debug")

	t.Run("Debug", func(t *testing.T) {
		l.Debug("test")
	})

	t.Run("Debugf", func(t *testing.T) {
		l.Debugf("test %d", 123)
	})

	t.Run("Info", func(t *testing.T) {
		l.Info("test")
	})

	t.Run("Infof", func(t *testing.T) {
		l.Infof("test %d", 123)
	})

	t.Run("Warn", func(t *testing.T) {
		l.Warn("test")
	})

	t.Run("Warnf", func(t *testing.T) {
		l.Warnf("test %d", 123)
	})

	t.Run("Error", func(t *testing.T) {
		l.Error("test")
	})

	t.Run("Errorf", func(t *testing.T) {
		l.Errorf("test %d", 123)
	})
}

// Note: We cannot test Fatal/Fatalf as they call os.Exit(1)
