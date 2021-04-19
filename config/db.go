package config

import (
"context"
"fmt"
"go.mongodb.org/mongo-driver/bson"
"go.mongodb.org/mongo-driver/mongo"
"go.mongodb.org/mongo-driver/mongo/options"
"go.mongodb.org/mongo-driver/mongo/readpref"
"log"
"os"
"time"
)

var CTX context.Context
var UserCollection *mongo.Collection
var SessionCollection *mongo.Collection
var BookmarkCollection *mongo.Collection

func init() {
	// get a mongo sessions
	// connecting to mongodb with authentication.
	databaseURI := os.Getenv("DATABASE_URI")
	client, err := mongo.NewClient(options.Client().ApplyURI(databaseURI))
	if err != nil {
		log.Fatal(err)
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	CTX = ctx
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatal(err)
	}
	databases, err := client.ListDatabaseNames(ctx, bson.M{})

	fmt.Println(databases)
	// BookmarkDB := client.Database("BookmarkDB")
	BookmarkCollection = client.Database("BookmarkDB").Collection("Bookmarks")
	UserCollection = client.Database("BookmarkDB").Collection("Users")
	SessionCollection = client.Database("BookmarkDB").Collection("Sessions")
	fmt.Println(BookmarkCollection)
	fmt.Println(UserCollection)
	fmt.Println(SessionCollection)
}

