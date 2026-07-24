package middleware

import (
	"os"
	"testing"
)

func TestConfigureJWTSecretRejectsMissingOrWeakSecret(t *testing.T) {
	original := os.Getenv("JWT_SECRET")
	t.Cleanup(func() { os.Setenv("JWT_SECRET", original) })

	for _, secret := range []string{"", "short"} {
		os.Setenv("JWT_SECRET", secret)
		if err := ConfigureJWTSecret(); err == nil {
			t.Fatalf("expected secret %q to be rejected", secret)
		}
	}
}

func TestConfigureJWTSecretAcceptsStrongSecret(t *testing.T) {
	original := os.Getenv("JWT_SECRET")
	t.Cleanup(func() { os.Setenv("JWT_SECRET", original) })
	secret := "12345678901234567890123456789012"
	os.Setenv("JWT_SECRET", secret)
	if err := ConfigureJWTSecret(); err != nil {
		t.Fatalf("expected strong secret to be accepted: %v", err)
	}
	if string(JWTSecret) != secret {
		t.Fatal("configured secret was not stored")
	}
}
