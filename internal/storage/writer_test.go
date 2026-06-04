package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDirIfNotExist(t *testing.T) {
	t.Run("creates nested directories", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "a", "b", "c")
		require.NoError(t, CreateDirIfNotExist(dir))
		info, err := os.Stat(dir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("no error if directory already exists", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, CreateDirIfNotExist(dir))
	})
}

func TestAppendContentToNote(t *testing.T) {
	t.Run("creates new file with content", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "note.md")
		require.NoError(t, AppendContentToNote(p, "hello", "", "", false))
		got, _ := os.ReadFile(p)
		assert.Equal(t, "hello", string(got))
	})

	t.Run("creates new file with heading prefix", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "note.md")
		require.NoError(t, AppendContentToNote(p, "body", "## H2", "", false))
		got, _ := os.ReadFile(p)
		assert.Equal(t, "## H2\nbody", string(got))
	})

	t.Run("appends to existing file", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "note.md")
		os.WriteFile(p, []byte("first"), 0o644)
		require.NoError(t, AppendContentToNote(p, "second", "", "\n\n***\n\n", false))
		got, _ := os.ReadFile(p)
		assert.Equal(t, "first\n\n***\n\nsecond", string(got))
	})

	t.Run("prepends with reversedOrder", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "note.md")
		os.WriteFile(p, []byte("old"), 0o644)
		require.NoError(t, AppendContentToNote(p, "new", "", "\n\n***\n\n", true))
		got, _ := os.ReadFile(p)
		assert.Equal(t, "new\n\n***\n\nold", string(got))
	})

	t.Run("custom delimiter", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "note.md")
		os.WriteFile(p, []byte("A"), 0o644)
		require.NoError(t, AppendContentToNote(p, "B", "", "\n---\n", false))
		got, _ := os.ReadFile(p)
		assert.Equal(t, "A\n---\nB", string(got))
	})

	t.Run("heading found appends after heading", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "note.md")
		os.WriteFile(p, []byte("# Title\nexisting content"), 0o644)
		require.NoError(t, AppendContentToNote(p, "new", "# Title", "\n\n***\n\n", false))
		got, _ := os.ReadFile(p)
		assert.Equal(t, "# Title\nexisting content\n\n***\n\nnew", string(got))
	})

	t.Run("heading found prepends after heading with reversedOrder", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "note.md")
		os.WriteFile(p, []byte("# Title\nexisting content"), 0o644)
		require.NoError(t, AppendContentToNote(p, "new", "# Title", "\n\n***\n\n", true))
		got, _ := os.ReadFile(p)
		assert.Equal(t, "# Title\nnew\n\n***\n\nexisting content", string(got))
	})

	t.Run("heading not found prepends heading to content then appends", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "note.md")
		os.WriteFile(p, []byte("some text"), 0o644)
		require.NoError(t, AppendContentToNote(p, "body", "## Missing", "\n\n***\n\n", false))
		got, _ := os.ReadFile(p)
		assert.Equal(t, "some text\n\n***\n\n## Missing\nbody", string(got))
	})

	t.Run("creates parent directories", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "sub", "dir", "note.md")
		require.NoError(t, AppendContentToNote(p, "data", "", "", false))
		got, _ := os.ReadFile(p)
		assert.Equal(t, "data", string(got))
	})
}

func TestSanitizeFileName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"hello.md", "hello.md"},
		{`a\b/c:d*e?f"g<h>i|j`, "abcdefghij"},
		{"line\none\rtwo", "lineonetwo"},
		{"normal-name_v2", "normal-name_v2"},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, SanitizeFileName(tc.input), "input: %q", tc.input)
	}
}

func TestSanitizeFilePath(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"dir/file.md", "dir/file.md"},
		{"a/b\\c:d*e?f\"g<h>i|j", "a/bcdefghij"},
		{"path/to/note", "path/to/note"},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, SanitizeFilePath(tc.input), "input: %q", tc.input)
	}
}

func TestGetUniqueFilePath(t *testing.T) {
	t.Run("returns basePath when no collision", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "note.md")
		result := GetUniqueFilePath(p, time.Now(), "md", nil)
		assert.Equal(t, p, result)
	})

	t.Run("generates unique path on filesystem collision", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "note.md")
		os.WriteFile(p, []byte("x"), 0o644)

		date := time.Date(2024, 3, 15, 10, 30, 45, 123000000, time.UTC)
		result := GetUniqueFilePath(p, date, "md", nil)

		assert.NotEqual(t, p, result)
		assert.Contains(t, result, "note - 20240315103045123.md")
	})

	t.Run("respects existingPaths map", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "note.md")
		existing := map[string]bool{p: true}

		date := time.Date(2024, 3, 15, 10, 30, 45, 123000000, time.UTC)
		result := GetUniqueFilePath(p, date, "md", existing)

		assert.NotEqual(t, p, result)
		assert.Contains(t, result, "note - 20240315103045123.md")
	})

	t.Run("increments on double collision", func(t *testing.T) {
		dir := t.TempDir()
		base := filepath.Join(dir, "note.md")
		os.WriteFile(base, []byte("x"), 0o644)

		date := time.Date(2024, 3, 15, 10, 30, 45, 123000000, time.UTC)
		first := filepath.Join(dir, "note - 20240315103045123.md")
		os.WriteFile(first, []byte("x"), 0o644)

		result := GetUniqueFilePath(base, date, "md", nil)
		assert.Contains(t, result, "note - 20240315103045124.md")
	})
}
