package main

import (
	"context"
	"errors"
	"time"

	"github.com/kamva/mgm/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ObjectSharingSession struct {
	mgm.DefaultModel `bson:",inline"`
	TTL              time.Duration          `json:"ttl"`
	ExpiryDate       time.Time              `bson:"expiry_date" json:"expiry_date"`
	Metadata         map[string]interface{} `json:"metadata"`
	OUUID            string                 `json:"ouuid"`
}

func (s *ObjectSharingSession) CreateIndex() error {
	col := mgm.Coll(s)
	_, err := col.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.M{"ouuid": 1},
		Options: options.MergeIndexOptions(
			options.Index().SetUnique(true),
			options.Index().SetName("ouuid"),
		),
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *ObjectSharingSession) Create() error {
	return CreateSession(s)
}

func CreateSession(s *ObjectSharingSession) error {
	// Remove all previous sessions
	if err := removeUnexpired(s.OUUID); err != nil {
		return err
	}
	col := mgm.Coll(s)
	if err := col.Create(s, nil); err != nil {
		return err
	}
	return nil
}

// Remove all previous sessions
func removeUnexpired(ouuid string) error {
	col := mgm.Coll(&ObjectSharingSession{})
	_, err := col.DeleteMany(
		context.Background(),
		bson.M{
			"ouuid": ouuid,
			"expiry_date": bson.M{
				"$gt": time.Now(),
			},
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func (s *ObjectSharingSession) CheckExpiration() bool {
	return s.ExpiryDate.Before(time.Now())
}

// fetch object latest session
func FetchLatestSession(ouuid string) (*ObjectSharingSession, error) {
	col := mgm.Coll(&ObjectSharingSession{})
	var ss []ObjectSharingSession

	crs, err := col.Find(
		context.Background(),
		bson.M{"ouuid": ouuid},
		options.Find().SetSort(bson.M{
			"expiry_date": -1,
		}).SetLimit(1),
	)
	if err != nil {
		return nil, err
	}

	if err := crs.All(context.Background(), &ss); err != nil {
		return nil, err
	}

	return &ss[0], nil
}

// fetch object session with session id
func FetchSession(id string) (*ObjectSharingSession, error) {
	col := mgm.Coll(&ObjectSharingSession{})
	var s ObjectSharingSession

	if id == "" {
		return nil, errors.New("session id is empty")
	}

	if err := col.FindByID(id, &s); err != nil {
		return nil, err
	}

	return &s, nil
}

func (s *ObjectSharingSession) BelongToObj(uuid string) (bool, error) {
	return s.OUUID == uuid, nil
}
