package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/kamva/mgm/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Resource struct {
	mgm.DefaultModel `bson:",inline"`
	UUID             string             `json:"uuid"`
	Name             string             `json:"name"`
	Bucket           primitive.ObjectID `json:"bucket_id" bson:"bucket_id"`
	Key              string             `json:"key"`

	Metadata Metadata `json:"metadata"`
}

type ObjectFile struct {
	Bucket   string
	File     *os.File
	Object   *Object
	Resource *Resource
}

func (r *Resource) CreateIndexes() error {
	col := mgm.Coll(r)
	_, err := col.Indexes().CreateMany(
		context.Background(),
		[]mongo.IndexModel{
			{
				Keys: bson.M{"uuid": 1},
				Options: options.MergeIndexOptions(
					options.Index().SetUnique(true),
				),
			},
			{
				Keys: bson.M{"name": 1},
				Options: options.MergeIndexOptions(
					options.Index().SetUnique(true),
				),
			},
			{
				Keys:    bson.M{"bucket_id": 1},
				Options: nil,
			},
		},
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *Resource) PopulateBucket() error {
	bkt, err := r.GetBucket()
	if err != nil {
		return err
	}
	r.Bucket = bkt.ID
	return nil
}

func (r *Resource) GetBucket() (*Bucket, error) {
	bkt, err := FetchBucketByID(r.Bucket)
	if err != nil {
		return nil, err
	}
	if bkt == nil {
		return nil, ErrBucketNotFound
	}

	if bkt.ID != r.Bucket {
		return nil, ErrBucketNotFound
	}

	return bkt, nil
}

func (r *Resource) S3Path() (string, error) {
	bkt, err := r.GetBucket()
	if err != nil {
		return "", err
	}
	s3 := S3Path{
		Bucket: bkt.Name,
		Paths:  []string{r.Key},
	}
	return s3.String(), nil
}

func (r *Resource) Path() string {
	bkt, err := r.GetBucket()
	if err != nil {
		return ""
	}
	return filepath.Join(bkt.Name, r.Key)
}

func (r *Resource) Create(bkt string) error {
	if err := r.PopulateBucket(); err != nil {
		return err
	}
	return FindOrCreateResource(r)
}

func FindOrCreateResource(r *Resource) error {
	// check if it exists
	found, err := FindResource(r)
	if err != nil {
		return err
	}
	// if it does, return it
	if found != nil {
		*r = *found
		return nil
	}
	// if it doesn't, create it
	return createResource(r)
}

func FindWithS3(s3 *S3Path, r *Resource) error {
	bkt, err := FetchBucket(s3.Bucket)
	if err != nil {
		return err
	}
	q := &Resource{
		Bucket: bkt.ID,
		Name:   s3.Paths[0],
	}
	rsrc, err := FindResource(q)
	if err != nil {
		return err
	}
	if rsrc == nil {
		return errors.New("resource not found")
	}
	*r = *rsrc
	return nil
}

func Find(r *Resource) error {
	rsrc, err := FindResource(r)
	if err != nil {
		return err
	}
	if rsrc == nil {
		return errors.New("resource not found")
	}
	*r = *rsrc
	return nil
}

func FindResource(query *Resource) (*Resource, error) {
	r := &Resource{}

	fltr, err := buildFilter(query)
	if err != nil {
		return nil, err
	}

	col := mgm.Coll(r)

	res := col.FindOne(context.Background(), fltr)
	fErr := res.Err()

	if errors.Is(fErr, mongo.ErrNoDocuments) {
		return nil, nil
	}

	if fErr != nil {
		return nil, fErr
	}

	if err := res.Decode(r); err != nil {
		return nil, err
	}

	return r, nil
}

func GetResourceID(key string) (*primitive.ObjectID, error) {
	p := strings.Split(key, "/")
	n := p[len(p)-1]

	r := &Resource{}
	err := mgm.Coll(r).First(
		bson.M{"name": n},
		r,
		options.FindOne().SetProjection(bson.M{"_id": 1}),
	)

	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrResourceNotFound
	}

	if err != nil {
		return nil, err
	}
	return &r.ID, nil
}

func FindResourceByID(ID primitive.ObjectID) (*Resource, error) {
	r := &Resource{}
	err := mgm.Coll(&Resource{}).FindByID(ID, r)

	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return r, nil
}

func createResource(r *Resource) error {
	// Validate bucket existence
	if _, err := FetchBucketByID(r.Bucket); err != nil {
		return err
	}

	if r.Name == "" {
		return errors.New("name is required")
	}

	// 0- Create UUID
	uuid, _ := uuid.NewRandom()
	r.UUID = uuid.String()

	// 1- create directory in bucket
	if _, err := CreateDir(r.Path()); err != nil {
		return err
	}

	// 2- create resource in database
	if err := mgm.Coll(r).Create(r); err != nil {
		return err
	}
	return nil
}

func buildFilter(r *Resource) (bson.M, error) {
	fltr := bson.M{}
	if r.UUID != "" {
		fltr["uuid"] = r.UUID
	}
	if r.Name != "" {
		fltr["name"] = r.Name
	}

	if len(fltr) == 0 {
		return nil, errors.New("no filter found to find resource")
	}

	return fltr, nil
}

func (r *Resource) Delete(force bool) error {
	return deleteResource(r, force)
}

func deleteResource(r *Resource, force bool) error {
	rsrc, err := FindResource(r)
	if err != nil {
		return err
	}
	if rsrc == nil {
		return errors.New("resource not found")
	}

	if !force {
		// check count of files in directory
		empty, err := IsEmptyDir(r.Path())
		if err != nil {
			return err
		}
		if !empty {
			return errors.New("resource is not empty")
		}
	}

	if err := DeleteDir(rsrc.Path(), force); err != nil {
		return err
	}

	// delete resource from database
	col := mgm.Coll(rsrc)
	_, err = col.DeleteOne(context.Background(), bson.M{"uuid": rsrc.UUID})
	if err != nil {
		return err
	}
	return nil
}

func (r *Resource) ListFiles() ([]*os.File, error) {
	return listFiles(r)
}

func listFiles(r *Resource) ([]*os.File, error) {
	return ListFilesInDir(r.Path())
}

func (r *Resource) Exists() (bool, error) {
	return resourceExists(r)
}

func resourceExists(r *Resource) (bool, error) {
	found, err := FindResource(r)
	if err != nil {
		return false, err
	}

	if found == nil {
		return false, nil
	}
	return true, nil
}

func ListObjectsS3(s3 *S3Path) ([]*ObjectFile, error) {
	r := &Resource{}
	if err := FindWithS3(s3, r); err != nil {
		return nil, err
	}

	files, err := ListFilesInDir(r.Path())
	if err != nil {
		return nil, err
	}
	defer CloseFiles(files)

	objects := mapToObjectFiles(files, r)
	return objects, nil
}

func mapToObjectFiles(files []*os.File, r *Resource) []*ObjectFile {
	objects := []*ObjectFile{}
	for _, f := range files {
		// find object file in database
		objects = append(objects, &ObjectFile{
			File:     f,
			Bucket:   "-",
			Resource: r,
			Object:   &Object{},
		})
	}
	return objects
}

func GetObject(r *Resource, filename string) (*ObjectFile, error) {
	return nil, nil
}
