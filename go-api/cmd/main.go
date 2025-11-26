package main

import (
	"log"

	"github.com/joho/godotenv"

	"julianmorley.ca/con-plar/prog2270/internal/router"
	"julianmorley.ca/con-plar/prog2270/pkg/global"
	"julianmorley.ca/con-plar/prog2270/pkg/mongo"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	mongo.InitMongoDB()
	mongo.EnsureIndexesOnStartup()
	router.InitEngine()
	router.InitializeRoutes()

	port := global.GetEnvOrDefault("PORT", "8000")
	log.Printf("Server is running on port %s", port)

	if err := router.Router.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
