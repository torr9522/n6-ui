package config

import "testing"

func TestEmbeddedRuntimeVersion(t *testing.T) {
	if got, want := GetVersion(), "v0.1.0"; got != want {
		t.Fatalf("GetVersion() = %q, want %q", got, want)
	}
	if got, want := GetXrayRuntimeVersion(), "26.5.3"; got != want {
		t.Fatalf("GetXrayRuntimeVersion() = %q, want %q", got, want)
	}
}
