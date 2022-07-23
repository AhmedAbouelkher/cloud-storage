package main

import (
	"github.com/kamva/mgm/v3"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	database = "storage"
)

func OpenDBConnection() error {
	err := mgm.SetDefaultConfig(
		nil,
		database,
		options.Client().ApplyURI("mongodb://127.0.0.1:27017"),
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
	return nil
}
