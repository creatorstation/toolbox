package db

import (
	"context"
	"fmt"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var mongoDatabase *mongo.Database

// ConnectMongo establishes a connection to MongoDB
func ConnectMongo() {
	ctx := context.Background()

	opts := options.Client().ApplyURI(os.Getenv("MONGO_URI"))
	mongoClient, err := mongo.Connect(ctx, opts)

	if err != nil {
		println("mongo.Connect failed")
		fmt.Println(err)

		return
	}

	mongoDatabase = mongoClient.Database("story_harvester")
}

// GetMongoDB returns the MongoDB database
func GetMongoDB() *mongo.Database {
	return mongoDatabase
}
