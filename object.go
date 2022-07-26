package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kamva/mgm/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Object struct {
	mgm.DefaultModel `bson:",inline"`

	UUID       string `json:"uuid"`
	Title      string `json:"title"`
	Type       string `json:"type"`
	Size       int    `json:"size"`
	Directory  string `json:"directory"`
	BucketName string `json:"bucket_name"`
}

func (o *Object) CreateIndex() error {
	col := mgm.Coll(o)
	_, err := col.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.M{"uuid": 1},
		Options: options.MergeIndexOptions(
			options.Index().SetUnique(true),
			options.Index().SetName("uuid"),
		),
	})
	if err != nil {
		return err
	}
	return nil
}

type SaveConfig struct {
	BucketID string
	Reader   io.Reader
	Key      string
}

func (o *Object) Save(cfg *SaveConfig) (string, error) {
	return SaveObject(o, cfg)
}

func SaveObject(o *Object, cfg *SaveConfig) (string, error) {
	// Validate bucket existence
	bkt, err := FetchBucket(cfg.BucketID)
	if err != nil {
		return "", err
	}

	// Create uuid
	uuid, _ := uuid.NewRandom()
	o.UUID = uuid.String()

	// parse path to handle sub directories in bucket
	k := cfg.Key
	if k == "" {
		k = o.Title
	}

	t := uuid.String() + filepath.Ext(k)
	o.Title = t

	dir := filepath.Dir(cfg.Key)
	p := filepath.Join(bkt.Name, dir, t) // bucket/new/image.jpg

	buf := new(bytes.Buffer)
	buf.ReadFrom(cfg.Reader)

	f, err := CreateFile(p, buf.Bytes())
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Update object
	o.Size = int(buf.Len())
	o.BucketName = bkt.Name

	if dir != "." {
		dir = filepath.Join(bkt.Name, dir)
	}
	o.Directory = dir

	// Store object
	if err := mgm.Coll(o).Create(o); err != nil {
		return "", err
	}
	return uuid.String(), nil
}

// Fetch object by uuid
func FetchObject(uuid string) (*Object, error) {
	o := &Object{}
	if err := mgm.Coll(o).FindOne(context.Background(), bson.M{"uuid": uuid}).Decode(o); err != nil {
		return nil, err
	}
	return o, nil
}

func DeleteObject(uuid string) error {
	// Fetch metadata from database
	o := &Object{}
	if err := mgm.Coll(o).FindOne(
		context.Background(),
		bson.M{"uuid": uuid},
	).Decode(o); err != nil {
		return err
	}

	// Delete file
	p := filepath.Join(o.BucketName, o.Directory, o.Title)
	if err := DeleteFile(p); err != nil {
		return err
	}

	// Delete object
	if _, err := mgm.Coll(o).DeleteOne(
		context.Background(),
		bson.M{"uuid": uuid},
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
	// build http://localhost:8000/share/<bucket>/<uuid>?ttl=<ttl>

	// Validate if bucket exists
	bkt, err := FetchBucket(o.BucketName)
	if err != nil {
		return "", nil, err
	}

	ttl := shr.TTL
	if ttl == 0 {
		ttl = time.Duration(3600) // 1 minute
	}

	// Generate sharable session
	session := &ObjectSharingSession{
		OUUID:      o.UUID,
		TTL:        ttl,
		ExpiryDate: CalculateExpiration(ttl),
	}
	if err := CreateSession(session); err != nil {
		return "", nil, err
	}

	l := fmt.Sprintf(
		"/share/%s/%s?ttl=%d&session=%s",
		bkt.Name,
		o.Title,
		ttl,
		session.ID.Hex(),
	)
	return JoinUrl(l), session, nil
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

var (
	ErrSessionExpired  = errors.New("session expired")
	ErrSessionNotFound = errors.New("session not found")
)

// Serve object from local filesystem
func ServeObject(uuid string, sn string) (*ServedFile, error) {
	// fetch object latest sharing session
	if err := checkSession(uuid, sn); err != nil {
		return nil, err
	}

	// Fetch metadata from database
	o := &Object{}
	if err := mgm.Coll(o).FindOne(
		context.Background(),
		bson.M{"uuid": uuid},
	).Decode(o); err != nil {
		return nil, err
	}

	// Serve object
	p := filepath.Join(o.BucketName, o.Directory, o.Title)
	f, err := GetFile(p)
	if err != nil {
		return nil, err
	}
	return &ServedFile{
		File: f,
		Type: o.Type,
	}, nil
}

func checkSession(uuid string, sn string) error {
	// fetch object latest sharing session
	s, err := FetchSession(sn)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return ErrSessionExpired
		}
		return err
	}
	if bgs, _ := s.BelongToObj(uuid); !bgs {
		return ErrSessionNotFound
	}

	if s.CheckExpiration() {
		return ErrSessionExpired
	}
	return nil
}

var (
	AllowedObjectTypes = []string{
		"^image/*",
		"^video/*",
		"^audio/*",
		"^application/pdf",
		"^application/vnd.apple.mpegurl",
		"^application/dash+xml",
	}
)

func CheckType(t string) bool {
	rgx := regexp.MustCompile(constructRegex())
	return rgx.MatchString(t)
}

func constructRegex() string {
	return strings.Join(AllowedObjectTypes, "|")
}
