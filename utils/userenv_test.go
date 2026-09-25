package utils

import (
	"slices"
	"testing"
)

func TestSecretEnvNeverReachesChildren(t *testing.T) {
	base := []string{"PATH=/bin", "TYPSTIFY_SERVER_PASSWORD=hunter2", "HOME=/data"}

	merged := MergeEnv(base, map[string]string{"TYPSTIFY_SERVER_PASSWORD": "from-login-shell"})
	for _, kv := range merged {
		if len(kv) >= 24 && kv[:24] == "TYPSTIFY_SERVER_PASSWORD" {
			t.Fatalf("MergeEnv leaked %q", kv)
		}
	}

	scrubbed := ScrubSecretEnv(base)
	if !slices.Equal(scrubbed, []string{"PATH=/bin", "HOME=/data"}) {
		t.Fatalf("ScrubSecretEnv = %v", scrubbed)
	}
}
