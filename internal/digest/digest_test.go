package digest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCalculate(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "empty string",
			content: "",
			want:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:    "hello world",
			content: "hello world",
			want:    "sha256:b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
		{
			name:    "deterministic - same input same output",
			content: "test content",
			want:    "sha256:6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Calculate(tt.content)
			if got != tt.want {
				t.Errorf("Calculate(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestCalculateDeterminism(t *testing.T) {
	content := "some arbitrary content for testing"
	first := Calculate(content)
	second := Calculate(content)

	if first != second {
		t.Errorf("Calculate is not deterministic: first=%q, second=%q", first, second)
	}
}

func TestMatch(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
		want     bool
	}{
		{
			name:     "matching digest",
			content:  "hello world",
			expected: "sha256:b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
			want:     true,
		},
		{
			name:     "non-matching digest",
			content:  "hello world",
			expected: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
			want:     false,
		},
		{
			name:     "empty content matching",
			content:  "",
			expected: "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Match(tt.content, tt.expected)
			if got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.content, tt.expected, got, tt.want)
			}
		})
	}
}

func TestShort(t *testing.T) {
	tests := []struct {
		name   string
		digest string
		want   string
	}{
		{
			name:   "standard digest",
			digest: "sha256:b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
			want:   "b94d27b9934d",
		},
		{
			name:   "another digest",
			digest: "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			want:   "e3b0c44298fc",
		},
		{
			name:   "short input - less than 19 chars returns as-is",
			digest: "sha256:abc",
			want:   "sha256:abc",
		},
		{
			name:   "exactly 19 chars - returns as-is (at boundary)",
			digest: "sha256:123456789012",
			want:   "sha256:123456789012",
		},
		{
			name:   "20 chars - just over boundary, gets shortened",
			digest: "sha256:1234567890123",
			want:   "123456789012",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Short(tt.digest)
			if got != tt.want {
				t.Errorf("Short(%q) = %q, want %q", tt.digest, got, tt.want)
			}
		})
	}
}

func TestCalculateFile(t *testing.T) {
	t.Run("existing file", func(t *testing.T) {
		// Create a temp file
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.txt")
		content := "file content for testing"

		if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		got, err := CalculateFile(tmpFile)
		if err != nil {
			t.Errorf("CalculateFile() error = %v", err)
			return
		}

		want := Calculate(content)
		if got != want {
			t.Errorf("CalculateFile() = %q, want %q", got, want)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := CalculateFile("/nonexistent/path/file.txt")
		if err == nil {
			t.Error("CalculateFile() expected error for non-existent file, got nil")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "empty.txt")

		if err := os.WriteFile(tmpFile, []byte{}, 0644); err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		got, err := CalculateFile(tmpFile)
		if err != nil {
			t.Errorf("CalculateFile() error = %v", err)
			return
		}

		want := Calculate("")
		if got != want {
			t.Errorf("CalculateFile() = %q, want %q", got, want)
		}
	})
}

func TestMatchFile(t *testing.T) {
	t.Run("matching file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.txt")
		content := "matching content"

		if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		digest := Calculate(content)
		match, err := MatchFile(tmpFile, digest)
		if err != nil {
			t.Errorf("MatchFile() error = %v", err)
			return
		}
		if !match {
			t.Error("MatchFile() = false, want true")
		}
	})

	t.Run("non-matching file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.txt")

		if err := os.WriteFile(tmpFile, []byte("actual content"), 0644); err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		wrongDigest := Calculate("different content")
		match, err := MatchFile(tmpFile, wrongDigest)
		if err != nil {
			t.Errorf("MatchFile() error = %v", err)
			return
		}
		if match {
			t.Error("MatchFile() = true, want false")
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := MatchFile("/nonexistent/path/file.txt", "sha256:abc")
		if err == nil {
			t.Error("MatchFile() expected error for non-existent file, got nil")
		}
	})
}
