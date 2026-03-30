package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	for _, k := range []string{"PORT", "DATABASE_PATH", "ASSETS_PATH", "CLIENT_PATH",
		"PLUGINS_PATH", "MIGRATIONS_DIR", "UPLOAD_MAX_SIZE_MB", "UPLOAD_ALLOWED_EXTS"} {
		t.Setenv(k, "")
	}

	cfg := Load()

	assert.Equal(t, "9024", cfg.Port)
	assert.Equal(t, "./devourer.db", cfg.DatabasePath)
	assert.Equal(t, "./assets", cfg.AssetsPath)
	assert.Equal(t, "./client", cfg.ClientPath)
	assert.Equal(t, "./plugins", cfg.PluginsPath)
	assert.Equal(t, "./migrations", cfg.MigrationsDir)
	assert.Equal(t, int64(1024), cfg.UploadMaxSizeMB)

	for _, ext := range []string{".epub", ".pdf", ".cbz", ".cbr"} {
		assert.True(t, cfg.UploadAllowedExts[ext], "expected default ext %s", ext)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("PORT", "8080")
	t.Setenv("DATABASE_PATH", "/data/test.db")
	t.Setenv("ASSETS_PATH", "/data/assets")
	t.Setenv("UPLOAD_MAX_SIZE_MB", "512")

	cfg := Load()

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "/data/test.db", cfg.DatabasePath)
	assert.Equal(t, "/data/assets", cfg.AssetsPath)
	assert.Equal(t, int64(512), cfg.UploadMaxSizeMB)
}

func TestLoad_InvalidIntFallsToDefault(t *testing.T) {
	t.Setenv("UPLOAD_MAX_SIZE_MB", "notanumber")

	cfg := Load()

	assert.Equal(t, int64(1024), cfg.UploadMaxSizeMB)
}

func TestParseExtList_NormalizesCase(t *testing.T) {
	m := parseExtList("EPUB,PDF,CBZ")

	require.True(t, m[".epub"])
	require.True(t, m[".pdf"])
	require.True(t, m[".cbz"])
	assert.False(t, m["EPUB"])
	assert.False(t, m["epub"])
}

func TestParseExtList_AddsDot(t *testing.T) {
	m := parseExtList("epub")

	assert.True(t, m[".epub"])
	assert.False(t, m["epub"])
}

func TestParseExtList_AlreadyHasDot(t *testing.T) {
	m := parseExtList(".epub")

	assert.True(t, m[".epub"])
	// Should not double-dot
	assert.False(t, m["..epub"])
}

func TestParseExtList_Empty(t *testing.T) {
	m := parseExtList("")

	assert.NotNil(t, m)
	assert.Empty(t, m)
}

func TestParseExtList_TrimsWhitespace(t *testing.T) {
	m := parseExtList(" epub , pdf ")

	assert.True(t, m[".epub"])
	assert.True(t, m[".pdf"])
}
