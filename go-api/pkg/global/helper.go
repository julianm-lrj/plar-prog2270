package global

import (
	"context"
	"log"
	"os"
	"time"
)

func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func GetDefaultTimer() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

func GetMongoURI() string {
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		log.Fatal("MONGODB_URI is not set in environment variables")
		os.Exit(1)
	}
	return mongoURI
}

func GetDatabaseName() string {
	dbName := GetEnvOrDefault("MONGODB_DATABASE", "plar_prog2270")
	return dbName
}
