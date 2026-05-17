package config

import (
	"encoding/hex"
	"log"
	"os"
	"strconv"
)

type Config struct {
	Port                   string
	DBURL                  string
	JWTSecret              string
	EncryptionKey          []byte // 32-byte AES-256 key, hex-encoded in env
	FilesDir               string // path inside the backend container where files are stored
	FilesHostDir           string // corresponding path on the Docker host (for bind mounts)
	IdleTimeoutMinutes     int
	CleanupIntervalMinutes int
}

func Load() *Config {
	encKeyHex := getEnv("ENCRYPTION_KEY", "")
	var encKey []byte
	if encKeyHex != "" {
		var err error
		encKey, err = hex.DecodeString(encKeyHex)
		if err != nil || len(encKey) != 32 {
			log.Fatal("ENCRYPTION_KEY must be a 64-char hex string (32 bytes)")
		}
	} else {
		// fallback for dev — not safe for production
		encKey = []byte("00000000000000000000000000000000")
	}

	return &Config{
		Port:                   getEnv("PORT", "8080"),
		DBURL:                  getEnv("DB_URL", "postgres://dockyard:dockyard@database:5432/dockyard"),
		JWTSecret:              getEnv("JWT_SECRET", "changeme"),
		EncryptionKey:          encKey,
		FilesDir:               getEnv("FILES_DIR", "/app/data/projects"),
		FilesHostDir:           getEnv("FILES_HOST_DIR", ""),
		IdleTimeoutMinutes:     getEnvInt("IDLE_TIMEOUT_MINUTES", 60),
		CleanupIntervalMinutes: getEnvInt("CLEANUP_INTERVAL_MINUTES", 5),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return n
}
