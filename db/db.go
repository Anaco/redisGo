package db

import (
	"context"
	"errors"

	"github.com/go-redis/redis"
)

//Database struct to wrap the redis client
type Database struct {
	Client *redis.Client
}

var (
	//ErrNil redis operation retuned a nil value.
	ErrNil = errors.New("no matching record found")
	//Ctx using TODO here, to setup and empty context
	Ctx = context.TODO()
)

//NewDatabase sets up a new connection, and checks it using Ping
func NewDatabase(address string) (*Database, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: "",
		DB:       0,
	})
	if err := client.Ping().Err(); err != nil {
		return nil, err
	}
	return &Database{
		Client: client,
	}, nil
}
