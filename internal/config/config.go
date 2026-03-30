package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port              string
	DatabasePath      string
	AssetsPath        string
	ClientPath        string
	PluginsPath       string
	MigrationsDir     string
	UploadMaxSizeMB   int64
	UploadAllowedExts map[string]bool
}

func Load() *Config {
	return &Config{
		Port:              getEnv("PORT", "9024"),
		DatabasePath:      getEnv("DATABASE_PATH", "./devourer.db"),
		AssetsPath:        getEnv("ASSETS_PATH", "./assets"),
		ClientPath:        getEnv("CLIENT_PATH", "./client"),
		PluginsPath:       getEnv("PLUGINS_PATH", "./plugins"),
		MigrationsDir:     getEnv("MIGRATIONS_DIR", "./migrations"),
		UploadMaxSizeMB:   getEnvInt64("UPLOAD_MAX_SIZE_MB", 1024),
		UploadAllowedExts: parseExtList(getEnv("UPLOAD_ALLOWED_EXTS", "epub,pdf,mobi,docx,doc,rtf,html,txt,cbz,cbr,zip,rar,7z,cb7")),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt64(key string, defaultVal int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return defaultVal
}

func parseExtList(s string) map[string]bool {
	m := make(map[string]bool)
	for _, ext := range strings.Split(s, ",") {
		ext = strings.TrimSpace(strings.ToLower(ext))
		if ext != "" {
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			m[ext] = true
		}
	}
	return m
}
