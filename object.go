package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/kamva/mgm/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrSessionExpired   = errors.New("session expired")
	ErrSessionNotFound  = errors.New("session not found")
	ErrBucketNotFound   = errors.New("bucket not found")
	ErrResourceNotFound = errors.New("resource not found")
)

type ObjectType struct {
	Ext  string `json:"ext"`
	Mime string `json:"mime"`
}

type Object struct {
	mgm.DefaultModel `bson:",inline"`

	EntityTag string `json:"entity_tag" bson:"entity_tag"`
	Title     string `json:"title"`

	Bucket   primitive.ObjectID `json:"bucket_id" bson:"bucket_id"`
	Resource primitive.ObjectID `json:"resource_id" bson:"resource_id"`

	Size int         `json:"size"`
	Type *ObjectType `json:"type"`

	Metadata map[string]interface{} `json:"metadata"`
}

func (o *Object) CreateIndexes() error {
	col := mgm.Coll(o)
	_, err := col.Indexes().CreateMany(
		context.Background(),
		[]mongo.IndexModel{
			{
				Keys: bson.M{"entity_tag": 1},
				Options: options.MergeIndexOptions(
					options.Index().SetUnique(true),
					options.Index().SetName("entity_tag"),
				),
			},
			{
				Keys: bson.M{"bucket_id": 1},
			},
			{
				Keys: bson.M{"resource_id": 1},
			},
		},
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (o *Object) GetBucket() (*Bucket, error) {
	b, err := FetchBucketByID(o.Bucket)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, ErrBucketNotFound
	}

	if b.ID != o.Bucket {
		return nil, ErrBucketNotFound
	}

	return b, nil
}

func (o *Object) GetResource() (*Resource, error) {
	r, err := FindResourceByID(o.Resource)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrResourceNotFound
	}

	if r.ID != o.Resource {
		return nil, ErrResourceNotFound
	}

	return r, nil
}

func (o *Object) GetKeyDetails() (*Bucket, *Resource, error) {
	bkt, err := o.GetBucket()
	if err != nil {
		return nil, nil, err
	}
	res, err := o.GetResource()
	if err != nil {
		return nil, nil, err
	}
	return bkt, res, nil
}

func (o *Object) Path(bkt *Bucket) (string, error) {
	bktN := bkt.Name
	if bktN == "" {
		b, err := o.GetBucket()
		if err != nil {
			return "", err
		}
		bktN = b.Name
	}
	k, err := o.Key()
	if err != nil {
		return "", err
	}
	return filepath.Join(bktN, k), nil
}

// key: resource/title.txt
func (o *Object) Key() (string, error) {
	r, err := o.GetResource()
	if err != nil {
		return "", err
	}
	return filepath.Join(r.Key, o.Title), nil
}

// s3: s3://bucket/resource/title.txt
func (o *Object) S3() (string, error) {
	b, r, err := o.GetKeyDetails()
	if err != nil {
		return "", err
	}
	s3 := &S3Path{
		Bucket:  b.Name,
		RawPath: filepath.Join(r.Key, o.Title),
	}
	return s3.String(), nil
}

type SaveConfig struct {
	Bucket   string
	Reader   io.Reader
	FilePath string
	Ext      string
	Mime     string
}

func (s *SaveConfig) Path() string {
	fp := filepath.Dir(s.FilePath)
	return filepath.Join(s.Bucket, fp)
}

func (o *Object) Save(cfg *SaveConfig) (string, error) {
	return SaveObject(o, cfg)
}

func SaveObject(o *Object, cfg *SaveConfig) (string, error) {
	// Validate bucket existence
	bkt, err := FetchBucket(cfg.Bucket)
	if err != nil {
		return "", err
	}

	obj, err := o.Create(cfg, bkt)
	if err != nil {
		return "", err
	}

	// Store object
	if err := mgm.Coll(o).Create(o); err != nil {
		return "", err
	}
	return obj.EntityTag, nil
}

func (o *Object) Create(cfg *SaveConfig, bkt *Bucket) (*Object, error) {
	return createObject(o, cfg, bkt)
}

func createObject(o *Object, cfg *SaveConfig, bkt *Bucket) (*Object, error) {
	// fetch bucket with id
	o.Bucket = bkt.ID
	o.Type = &ObjectType{}
	o.Type.Ext = cfg.Ext
	o.Type.Mime = cfg.Mime

	// Create uuid
	uuid, _ := uuid.NewRandom()
	o.EntityTag = uuid.String()

	// Saved file path construction
	sfp := ""

	fPath := cfg.FilePath // aka: file path name, aka: key

	// if key is not givin, then save directly to bucket
	o.Title = filepath.Base(fPath)

	// saved to bucket with key
	dir := filepath.Dir(fPath)
	if dir == "." {
		sfp = cfg.Path()
	} else {
		// create resource and get its directory
		r, err := createObjectResource(cfg)
		if err != nil {
			return nil, err
		}
		o.Resource = r.ID
		sfp = r.Path()
	}

	// Add file title to the path
	sfp = filepath.Join(sfp, o.Title)

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(cfg.Reader); err != nil {
		return nil, err
	}

	// Update object
	o.Size = buf.Len()

	f, err := CreateFile(sfp, buf.Bytes())
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return o, nil
}

func createObjectResource(cfg *SaveConfig) (*Resource, error) {
	k := filepath.Dir(cfg.FilePath) // resource key
	n := filepath.Base(k)           // resource name
	if n == "" {
		return nil, errors.New("resource name is empty")
	}
	r := &Resource{
		Bucket: cfg.Bucket,
		Name:   n,
		Key:    k,
	}
	if err := FindOrCreateResource(r); err != nil {
		return nil, err
	}
	return r, nil
}

// Fetch object by uuid
func FindObject(tag string) (*Object, error) {
	o := &Object{}
	if err := mgm.Coll(o).FindOne(
		context.Background(),
		bson.M{"entity_tag": tag},
	).Decode(o); err != nil {
		return nil, err
	}
	return o, nil
}

func DeleteObject(tag string) error {
	// Fetch metadata from database
	o := &Object{}
	if err := mgm.Coll(o).FindOne(
		context.Background(),
		bson.M{"entity_tag": tag},
	).Decode(o); err != nil {
		return err
	}

	p, err := o.Path(nil)
	if err != nil {
		return err
	}

	// Delete file
	if err := DeleteFile(p); err != nil {
		return err
	}

	// Delete object
	if _, err := mgm.Coll(o).DeleteOne(
		context.Background(),
		bson.M{"entity_tag": tag},
	); err != nil {
		return err
	}
	return nil
}

type ObjectShare struct {
	// Link expiration date in seconds
	TTL time.Duration

	Metadata map[string]interface{}
}

func (o *Object) GenerateSharableLink(shr *ObjectShare) (string, *ObjectSharingSession, error) {
	// Generate a link
	// build http://localhost:8000/share/<path/to/file>?ttl=<ttl>

	// Validate if bucket exists
	bkt, err := FetchBucketByID(o.Bucket)
	if err != nil {
		return "", nil, err
	}
	if bkt == nil {
		return "", nil, errors.New("bucket does not exist")
	}

	ttl := shr.TTL
	if ttl == 0 {
		ttl = time.Duration(3600) // 1 minute
	}

	// Generate sharable session
	session := &ObjectSharingSession{
		EntityTag:  o.EntityTag,
		TTL:        ttl,
		ExpiryDate: CalculateExpiration(ttl),
	}
	if err := CreateSession(session); err != nil {
		return "", nil, err
	}

	p, err := o.Path(bkt)
	if err != nil {
		return "", nil, err
	}

	uri, err := JoinUrl(fmt.Sprintf(
		"/share/%s?ttl=%d&session=%s",
		p,
		ttl,
		Base64Encode([]byte(session.ID.Hex())),
	))
	if err != nil {
		return "", nil, err
	}
	return uri, session, nil
}

func CalculateExpiration(ttl time.Duration) time.Time {
	return time.Now().Add(ttl * time.Second)
}

type ServedFile struct {
	File *os.File
	Type string
}

// close ServedFile
func (f *ServedFile) Close() error {
	return f.File.Close()
}

// show file name from served file
func (f *ServedFile) Name() string {
	return f.File.Name()
}

// Serve object from local filesystem
func ServeObject(tag string, sid string) (*ServedFile, error) {
	// fetch object latest sharing session
	id, _ := Base64Decode(sid)
	if err := checkSession(tag, string(id)); err != nil {
		return nil, err
	}

	// Fetch metadata from database
	o := &Object{}
	if err := mgm.Coll(o).FindOne(
		context.Background(),
		bson.M{"entity_tag": tag},
	).Decode(o); err != nil {
		return nil, err
	}

	p, err := o.Path(nil)
	if err != nil {
		return nil, err
	}

	// Serve object
	f, err := GetFile(p)
	if err != nil {
		return nil, err
	}
	return &ServedFile{
		File: f,
		Type: o.Type.Mime,
	}, nil
}

func checkSession(tag string, sid string) error {
	// fetch object latest sharing session
	s, err := FetchSession(sid)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return ErrSessionExpired
		}
		return err
	}
	if bgs, _ := s.BelongToObj(tag); !bgs {
		return ErrSessionNotFound
	}

	if s.CheckExpiration() {
		return ErrSessionExpired
	}
	return nil
}

type SaveMultipleConfig struct {
	BucketID string
	Objects  []*Object
	Configs  []*SaveConfig
}

func (s *SaveMultipleConfig) Save() ([]*Object, error) {
	return SaveMultipleObjects(context.Background(), s)
}

func (s *SaveMultipleConfig) Push(o *Object, cfg *SaveConfig) {
	s.Objects = append(s.Objects, o)
	s.Configs = append(s.Configs, cfg)
}

func SaveMultipleObjects(ctx context.Context, scfg *SaveMultipleConfig) ([]*Object, error) {
	bkt, err := FetchBucket(scfg.BucketID)
	if err != nil {
		return nil, err
	}

	var docs []interface{}
	var objs []*Object

	for i, o := range scfg.Objects {
		cfg := scfg.Configs[i]
		obj, err := o.Create(cfg, bkt)
		if err != nil {
			return nil, err
		}
		docs = append(docs, *obj)
		objs = append(objs, obj)
	}

	// Store objects
	if _, err := mgm.Coll(&Object{}).InsertMany(ctx, docs, nil); err != nil {
		return nil, err
	}

	return objs, nil
}

type UploadedObjectsResponse struct {
	Message string   `json:"message"`
	Objects []Object `json:"objects"`
	Bucket  string   `json:"bucket"`
	Subpath string   `json:"sub_path"`
	Path    string   `json:"path"`
}

func (u *UploadedObjectsResponse) FromObjects(objs []*Object) {
	u.Message = "objects created"
	for _, o := range objs {
		u.Objects = append(u.Objects, *o)
	}
}
