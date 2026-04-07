package runtime

import (
	"strings"
	"testing"
)

func TestAppleRuntime_Name(t *testing.T) {
	rt := NewApple(Stdio{})
	if rt.Name() != "Apple Containers" {
		t.Errorf("expected 'Apple Containers', got %q", rt.Name())
	}
}

func TestAppleRuntime_buildRunArgs(t *testing.T) {
	rt := NewApple(Stdio{})

	t.Run("basic args without hostname", func(t *testing.T) {
		args := rt.buildRunArgs(RunConfig{
			ContainerName: "my-container",
			ImageName:     "my-image:latest",
			HostPath:      "/home/user/project",
			WorkspacePath: "/project",
			Hostname:      "glovebox", // should be ignored
		})

		argsStr := strings.Join(args, " ")
		for _, want := range []string{
			"run", "-it",
			"--name my-container",
			"-v /home/user/project:/project",
			"-w /project",
		} {
			if !strings.Contains(argsStr, want) {
				t.Errorf("expected %q in args, got: %s", want, argsStr)
			}
		}

		// --hostname should NOT appear (Apple Containers uses --name for hostname)
		if strings.Contains(argsStr, "--hostname") {
			t.Error("Apple Containers should not use --hostname flag")
		}

		// Image should be last
		if args[len(args)-1] != "my-image:latest" {
			t.Errorf("expected image as last arg, got %q", args[len(args)-1])
		}
	})

	t.Run("env vars sorted deterministically", func(t *testing.T) {
		args := rt.buildRunArgs(RunConfig{
			ContainerName: "test",
			ImageName:     "test:latest",
			HostPath:      "/path",
			WorkspacePath: "/workspace",
			Env:           map[string]string{"ZZZ": "last", "AAA": "first"},
		})

		argsStr := strings.Join(args, " ")
		if !strings.Contains(argsStr, "-e AAA=first") {
			t.Error("expected AAA env var in args")
		}
		if !strings.Contains(argsStr, "-e ZZZ=last") {
			t.Error("expected ZZZ env var in args")
		}

		// Verify AAA comes before ZZZ
		aIdx := strings.Index(argsStr, "AAA")
		zIdx := strings.Index(argsStr, "ZZZ")
		if aIdx > zIdx {
			t.Error("env vars should be sorted: AAA before ZZZ")
		}
	})
}

func TestAppleRuntime_Capabilities(t *testing.T) {
	rt := NewApple(Stdio{})
	caps := rt.Capabilities()

	if caps.SupportsDiff {
		t.Error("Apple Containers should not support diff")
	}
	if caps.SupportsCommit {
		t.Error("Apple Containers should not support commit")
	}
	if !caps.SupportsExport {
		t.Error("Apple Containers should support export")
	}
}

func TestAppleRuntime_Diff_returnsErrNotSupported(t *testing.T) {
	rt := NewApple(Stdio{})
	_, err := rt.Diff("any-container")
	if err != ErrNotSupported {
		t.Errorf("expected ErrNotSupported, got %v", err)
	}
}

func TestAppleRuntime_Commit_returnsErrNotSupported(t *testing.T) {
	rt := NewApple(Stdio{})
	err := rt.Commit("any-container", "any-image")
	if err != ErrNotSupported {
		t.Errorf("expected ErrNotSupported, got %v", err)
	}
}

func TestAppleRuntime_normalizeExitError(t *testing.T) {
	rt := NewApple(Stdio{})

	t.Run("nil error passes through", func(t *testing.T) {
		if err := rt.normalizeExitError(nil); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})
}

func TestMatchesFilter(t *testing.T) {
	tests := []struct {
		name   string
		image  string
		filter string
		want   bool
	}{
		{"empty filter matches all", "anything:latest", "", true},
		{"exact match", "glovebox:base", "glovebox:base", true},
		{"name-only matches with tag", "glovebox:base", "glovebox", true},
		{"glob prefix match", "glovebox:base", "glovebox*", true},
		{"glob prefix no match", "other:latest", "glovebox*", false},
		{"no match", "other:latest", "glovebox", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesFilter(tt.image, tt.filter)
			if got != tt.want {
				t.Errorf("matchesFilter(%q, %q) = %v, want %v", tt.image, tt.filter, got, tt.want)
			}
		})
	}
}
