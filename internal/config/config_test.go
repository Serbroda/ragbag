package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUninterpolatedComposeValuesAreIgnored(t *testing.T) {
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("ADMIN_USERS", "${ADMIN_USERS:-}")
	t.Setenv("REGISTRATION_ENABLED", "false")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.AdminUsers) != 0 || cfg.RegistrationOpen {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestProductionNeedsASecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("RAGBAG_DEV", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected an error without JWT_SECRET")
	}
	t.Setenv("JWT_SECRET", "too-short")
	if _, err := Load(); err == nil {
		t.Fatal("expected an error for a short JWT_SECRET")
	}
}

func TestDotenvFilesNeverOverrideTheEnvironment(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(".env.local", "ADMIN_USERS=from-local\nDATABASE_URL=sqlite:local.db\n")
	write(".env", "ADMIN_USERS=from-env\nALLOW_PRIVATE_HOSTS=true\nDATABASE_URL=sqlite:env.db\n")
	t.Setenv("DATABASE_URL", "sqlite:real.db")
	// Cleared by t.Setenv/os.Unsetenv after the test
	for _, key := range []string{"ADMIN_USERS", "ALLOW_PRIVATE_HOSTS"} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}

	if err := LoadDotenv(); err != nil {
		t.Fatal(err)
	}
	for key, expected := range map[string]string{
		"DATABASE_URL":        "sqlite:real.db", // the real environment wins
		"ADMIN_USERS":         "from-local",     // .env.local wins over .env
		"ALLOW_PRIVATE_HOSTS": "true",           // .env fills the rest
	} {
		if actual := os.Getenv(key); actual != expected {
			t.Errorf("%s = %q, want %q", key, actual, expected)
		}
	}
}

func TestMissingDotenvFilesAreFine(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := LoadDotenv(); err != nil {
		t.Fatal(err)
	}
}
