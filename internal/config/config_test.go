package config

import "testing"

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
