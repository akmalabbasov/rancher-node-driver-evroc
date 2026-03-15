package rancherdriver

import "testing"

func TestSanitizeName(t *testing.T) {
	t.Parallel()

	got := sanitizeName("Rancher Node_01")
	if got != "rancher-node-01" {
		t.Fatalf("unexpected sanitized name: %q", got)
	}
}
