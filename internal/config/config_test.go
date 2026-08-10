package config

import "testing"

func TestLoad_Defaults(t *testing.T) {
	cfg := Load()
	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want %q", cfg.Addr, ":8080")
	}
	if cfg.DBPath != "./data/pixabros.db" {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, "./data/pixabros.db")
	}
	if cfg.DataDir != "./data" {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, "./data")
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("PIXABROS_ADDR", ":9090")
	t.Setenv("PIXABROS_DB_PATH", "/tmp/custom.db")
	t.Setenv("PIXABROS_DATA_DIR", "/tmp/custom-data")

	cfg := Load()
	if cfg.Addr != ":9090" {
		t.Errorf("Addr = %q, want %q", cfg.Addr, ":9090")
	}
	if cfg.DBPath != "/tmp/custom.db" {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, "/tmp/custom.db")
	}
	if cfg.DataDir != "/tmp/custom-data" {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, "/tmp/custom-data")
	}
}
