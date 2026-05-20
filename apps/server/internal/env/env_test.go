package env

import "testing"

func TestOrDefaultReturnsEnvValueWhenSet(t *testing.T) {
	t.Setenv("VELORA_TEST_ENV_KEY", "from-env")

	if got := OrDefault("VELORA_TEST_ENV_KEY", "fallback"); got != "from-env" {
		t.Fatalf("expected from-env, got %q", got)
	}
}

func TestOrDefaultReturnsFallbackWhenUnset(t *testing.T) {
	t.Setenv("VELORA_TEST_ENV_KEY", "")

	if got := OrDefault("VELORA_TEST_ENV_KEY", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}
}
