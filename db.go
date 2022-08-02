package main

import (
	"os"

	"github.com/kamva/mgm/v3"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func OpenDBConnection() error {
	db := os.Getenv("DB_NAME")
	uri := os.Getenv("MONGO_URL")
	err := mgm.SetDefaultConfig(
		nil,
		db,
		options.Client().ApplyURI(uri),
	)
	if err != nil {
		return err
	}

	processIndexes()

	return nil
}

func processIndexes() error {
	if err := (&Object{}).CreateIndex(); err != nil {
		return err
	}
	if err := (&ObjectSharingSession{}).CreateIndex(); err != nil {
		return err
	}
	if err := (&Bucket{}).CreateIndex(); err != nil {
		return err
	}
	return nil
}
