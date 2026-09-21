package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port string

	GCPProjectID string
	GCPLocation  string
	GeminiModel  string

	GCSBucketName string

	MongoURI      string
	MongoDatabase string

	AuthJWKSURL      string
	AuthPublicKeyPEM string
	Environment      string
}

func Load() *Config {
	port := getEnv("PORT", "8080")

	return &Config{
		Port:             port,
		GCPProjectID:     getEnv("GCP_PROJECT_ID", "swift-delight-441118-c4"),
		GCPLocation:      getEnv("GCP_LOCATION", "global"),
		GeminiModel:      getEnv("GEMINI_MODEL", "gemini-omni-flash-preview"),
		GCSBucketName:    getEnv("GCS_BUCKET_NAME", "frameverse-videos-gfh"),
		MongoURI:         os.Getenv("MONGO_URI"), // MongoURI:      getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDatabase:    getEnv("MONGO_DATABASE", "frameverse"),
		AuthJWKSURL:      getEnv("AUTH_JWKS_URL", "http://localhost:8081/.well-known/jwks.json"),
		AuthPublicKeyPEM: getEnv("AUTH_PUBLIC_KEY_PEM", ""),
		Environment:      getEnv("ENVIRONMENT", "development"),
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
