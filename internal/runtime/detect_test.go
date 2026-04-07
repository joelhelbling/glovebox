package runtime

import (
	"testing"
)

func TestDetect_withOverride(t *testing.T) {
	t.Run("docker override", func(t *testing.T) {
		result, err := Detect("docker", Stdio{})
		if err != nil {
			t.Skipf("Docker not available: %v", err)
		}
		if result.Runtime.Name() != "Docker" {
			t.Errorf("expected Docker runtime, got %q", result.Runtime.Name())
		}
		if result.FellBack {
			t.Error("explicit override should not be a fallback")
		}
	})

	t.Run("unknown override errors", func(t *testing.T) {
		_, err := Detect("nonexistent-runtime", Stdio{})
		if err == nil {
			t.Error("expected error for unknown runtime")
		}
	})
}

func TestDetect_autoDetection(t *testing.T) {
	result, err := Detect("", Stdio{})
	if err != nil {
		t.Skipf("No runtime available in test environment: %v", err)
	}
	if result.Runtime == nil {
		t.Error("expected a runtime to be detected")
	}
	if result.Runtime.Name() == "" {
		t.Error("expected runtime to have a name")
	}
}
