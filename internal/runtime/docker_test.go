package runtime

import (
	"strings"
	"testing"
)

func TestDockerRuntime_Name(t *testing.T) {
	rt := NewDocker(Stdio{})
	if rt.Name() != "Docker" {
		t.Errorf("expected 'Docker', got %q", rt.Name())
	}
}

func TestDockerRuntime_buildRunArgs(t *testing.T) {
	rt := NewDocker(Stdio{})

	t.Run("basic args", func(t *testing.T) {
		args := rt.buildRunArgs(RunConfig{
			ContainerName: "my-container",
			ImageName:     "my-image:latest",
			HostPath:      "/home/user/project",
			WorkspacePath: "/project",
			Hostname:      "glovebox",
		})

		argsStr := strings.Join(args, " ")
		for _, want := range []string{
			"run", "-it",
			"--name my-container",
			"-v /home/user/project:/project",
			"-w /project",
			"--hostname glovebox",
		} {
			if !strings.Contains(argsStr, want) {
				t.Errorf("expected %q in args, got: %s", want, argsStr)
			}
		}

		// Image should be last
		if args[len(args)-1] != "my-image:latest" {
			t.Errorf("expected image as last arg, got %q", args[len(args)-1])
		}
	})

	t.Run("env vars", func(t *testing.T) {
		args := rt.buildRunArgs(RunConfig{
			ContainerName: "test",
			ImageName:     "test:latest",
			HostPath:      "/path",
			WorkspacePath: "/workspace",
			Env:           map[string]string{"API_KEY": "secret", "FOO": "bar"},
		})

		argsStr := strings.Join(args, " ")
		if !strings.Contains(argsStr, "-e API_KEY=secret") {
			t.Error("expected API_KEY env var in args")
		}
		if !strings.Contains(argsStr, "-e FOO=bar") {
			t.Error("expected FOO env var in args")
		}
	})

	t.Run("empty hostname omitted", func(t *testing.T) {
		args := rt.buildRunArgs(RunConfig{
			ContainerName: "test",
			ImageName:     "test:latest",
			HostPath:      "/path",
			WorkspacePath: "/workspace",
			Hostname:      "",
		})

		argsStr := strings.Join(args, " ")
		if strings.Contains(argsStr, "--hostname") {
			t.Error("empty hostname should not produce --hostname flag")
		}
	})
}

func TestDockerRuntime_Capabilities(t *testing.T) {
	rt := NewDocker(Stdio{})
	caps := rt.Capabilities()

	if !caps.SupportsDiff {
		t.Error("Docker should support diff")
	}
	if !caps.SupportsCommit {
		t.Error("Docker should support commit")
	}
	if !caps.SupportsExport {
		t.Error("Docker should support export")
	}
}

func TestDockerRuntime_normalizeExitError(t *testing.T) {
	rt := NewDocker(Stdio{})

	t.Run("nil error passes through", func(t *testing.T) {
		if err := rt.normalizeExitError(nil); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})
}
