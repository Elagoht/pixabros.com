package config

import "os"

type Config struct {
	Addr    string
	DBPath  string
	DataDir string
}

func Load() Config {
	return Config{
		Addr:    getEnv("PIXABROS_ADDR", ":8080"),
		DBPath:  getEnv("PIXABROS_DB_PATH", "./data/pixabros.db"),
		DataDir: getEnv("PIXABROS_DATA_DIR", "./data"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
