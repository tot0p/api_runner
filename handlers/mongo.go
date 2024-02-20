package handlers

import (
	"context"
	"fmt"
	"github.com/tot0p/env"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type Item struct {
	containerID     string `bson:"containerID"`
	repositoryURL   string `bson:"repositoryURL"`
	creationDate    string `bson:"creationDate"`
	destructionDate string `bson:"destructionDate"`
}

func (i *Item) FromBSON(data bson.M) {
	i.containerID = data["containerID"].(string)
	i.repositoryURL = data["repositoryURL"].(string)
	i.creationDate = data["creationDate"].(string)
	i.destructionDate = data["destructionDate"].(string)
}

func (i *Item) ToBSON() bson.M {
	return bson.M{
		"containerID":     i.containerID,
		"repositoryURL":   i.repositoryURL,
		"creationDate":    i.creationDate,
		"destructionDate": i.destructionDate,
	}
}

var mongoClient *mongo.Client

func InitMongo() {
	mongoClient = CreateClient()
}

func CreateClient() *mongo.Client {
	err := env.Load()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	ctx := context.TODO()
	URI := env.Get("MONGODB_URI")
	fmt.Println(URI)

	loggerOptions := options.Logger().SetComponentLevel(options.LogComponentCommand, options.LogLevelDebug)
	clientOptions := options.Client().ApplyURI(URI).SetLoggerOptions(loggerOptions)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		fmt.Println(err, "Can't connect to mongo")
		return nil
	}
	if client != nil {
		err = client.Ping(ctx, nil)
		if err != nil {
			fmt.Println(err, "Can't ping mongo")
			return nil
		}
	}

	return client
}

func LogContainerCreation(containerID, repositoryUrl string) {
	time := time.Now()
	item := Item{
		containerID:     containerID,
		repositoryURL:   repositoryUrl,
		creationDate:    time.String(),
		destructionDate: "",
	}

	coll := mongoClient.Database("logs").Collection("docker")
	ctx := context.TODO()
	_, err := coll.InsertOne(ctx, item.ToBSON())
	if err != nil {
		fmt.Println("Can't log container creation", err)
	}

}

func LogContainerDestruction(containerID string) {
	time := time.Now().String()
	filter := bson.D{{"containerID", containerID}}
	update := bson.D{{"$set", bson.D{{"destructionDate", time}}}}
	coll := mongoClient.Database("logs").Collection("docker")
	ctx := context.TODO()
	_, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		fmt.Println("Can't log container destruction", err)
	}
}

func GetLogs() []map[string]interface{} {
	coll := mongoClient.Database("logs").Collection("docker")
	findOptions := options.Find()
	findOptions.SetLimit(5)
	cursor, err := coll.Find(context.TODO(), bson.D{}, findOptions)
	if err != nil {
		fmt.Println(err, "Unable to find logs")
	}
	var items []map[string]interface{}
	if err = cursor.All(context.TODO(), &items); err != nil {
		fmt.Println(err, "Error with cursor")
	}
	return items
}
