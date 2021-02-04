package db

import (
	"context"
	"errors"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

//DynamoDbClient struct for document client
type DynamoDbClient struct {
	Client    *dynamodb.Client
	TableName string
}

var (
	//ErrDdbNoResults thrown when a query returns empty items
	ErrDdbNoResults = errors.New("no results were found in query")
	//DDBCtx empty context
	DDBCtx = context.TODO()
)

//NewDynamoDbClient creates a new dynamoClient
func NewDynamoDbClient(tableName string) (*DynamoDbClient, error) {
	cfg, err := config.LoadDefaultConfig(DDBCtx, config.WithRegion("us-east-2"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
		return nil, err
	}

	svc := dynamodb.NewFromConfig(cfg)

	return &DynamoDbClient{
		Client:    svc,
		TableName: tableName,
	}, nil

}
