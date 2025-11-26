package mongo

import (
	"log"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"julianmorley.ca/con-plar/prog2270/pkg/global"
)

func GetMongoClient() *mongo.Client {
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)

	clientOptions := options.Client().ApplyURI(global.GetMongoURI()).SetServerAPIOptions(serverAPI)
	client, err := mongo.Connect(clientOptions)
	if err != nil {
		log.Fatalf("Failed to create MongoDB client: %v", err)
	}
	return client
}

func GetDatabase() *mongo.Database {
	return GetMongoClient().Database(global.GetDatabaseName())
}

func GetCollection(collectionName string) *mongo.Collection {
	return GetDatabase().Collection(collectionName)
}

func InitMongoDB() {

	client := GetMongoClient()
	ctx, cancel := global.GetDefaultTimer()
	defer cancel()

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	log.Println("Connected to MongoDB successfully")
}
