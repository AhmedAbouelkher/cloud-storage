package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/kamva/mgm/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	BucketCollection = "buckets"
)

type Bucket struct {
	mgm.DefaultModel `bson:",inline"`
	Name             string `json:"name"`
}

func (o *Bucket) CreateIndex() error {
	col := mgm.Coll(o)
	_, err := col.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.M{"name": 1},
		Options: options.MergeIndexOptions(
			options.Index().SetUnique(true),
			options.Index().SetName("name"),
		),
	})
	if err != nil {
		return err
	}
	return nil
}

// Create bucket
func (b *Bucket) Create() error {
	return CreateBucket(b)
}

func CreateBucket(b *Bucket) error {
	// Validate bucket existence
	if exists, err := Exists(b.Name); err != nil {
		if err != nil {
			return err
		} else if exists {
			return errors.New("bucket already exists")
		}
	}
	b.mutateName() // mutate name to avoid collision

	// Create bucket
	_, err := CreateDir(b.Name)
	if err != nil {
		return err
	}

	// Store bucket metadata
	if err := mgm.Coll(b).Create(b); err != nil {
		return err
	}
	return nil
}

func (b *Bucket) mutateName() error {
	n := normalizeName(b.Name)
	r, err := rand.Int(rand.Reader, big.NewInt(9999))
	if err != nil {
		return err
	}
	b.Name = fmt.Sprintf("%s-%d", n, r)
	return nil
}

func normalizeName(name string) string {
	// replace all special characters with underscore
	r := regexp.MustCompile(`[\s$&+,:;=?@#|'<>.^*()%!-]`)
	return strings.ToLower(r.ReplaceAllString(name, "_"))
}

// Delete bucket
func (b *Bucket) Delete() error {
	return DeleteBucket(b)
}

// DeleteBucket
func DeleteBucket(b *Bucket) error {
	// Validate bucket existence
	if exists, err := Exists(b.Name); err != nil {
		if err != nil {
			return err
		} else if !exists {
			return errors.New("bucket does not exist")
		}
	}

	// Delete bucket
	if err := DeleteDir(b.Name, false); err != nil {
		return err
	}

	// Delete bucket metadata
	if err := mgm.Coll(b).Delete(b); err != nil {
		return err
	}
	return nil
}

// Fetch bucket by name from database
func FetchBucket(name string) (*Bucket, error) {
	var b Bucket
	err := mgm.Coll(&Bucket{}).First(bson.M{"name": name}, &b)

	if err != nil {
		return nil, err
	}

	return &b, nil
}

func BucketExists(name string) (bool, error) {
	var b Bucket
	err := mgm.Coll(&Bucket{}).First(
		bson.M{"name": name},
		&b,
		options.FindOne().SetProjection(bson.M{"name": 1}),
	)

	if err != nil {
		return false, err
	}

	return true, err
}

func (b *Bucket) FetchObjects() ([]Object, error) {
	return FetchBucketObjects(b)
}

func FetchBucketObjects(b *Bucket) ([]Object, error) {
	var objects []Object
	cur, err := mgm.Coll(&Object{}).Find(
		context.Background(),
		bson.M{"bucketname": b.Name},
	)
	if err != nil {
		return nil, err
	}
	if err := cur.All(context.Background(), &objects); err != nil {
		return nil, err
	}

	return objects, nil

}
