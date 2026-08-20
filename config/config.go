package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port string

	GeminiAPIKey string
	GeminiModel  string

	GCSBucketName string

	MongoURI      string
	MongoDatabase string
}

func Load() *Config {
	port := getEnv("PORT", "8080")

	return &Config{
		Port:          port,
		GeminiAPIKey:  os.Getenv("GEMINI_API_KEY"),
		GeminiModel:   getEnv("GEMINI_MODEL", "gemini-omni-flash-preview"),
		GCSBucketName: os.Getenv("GCS_BUCKET_NAME"),
		MongoURI:      os.Getenv("MONGO_URI"),
		MongoDatabase: getEnv("MONGO_DATABASE", "frameverse"),
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	result, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return result
}
