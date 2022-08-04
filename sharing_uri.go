package main

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ShareUri struct {
	Session  string        `json:"session"`
	TTL      time.Duration `json:"ttl"`
	Bucket   string        `json:"bucket"`
	Key      string        `json:"key"`
	Metadata Metadata      `json:"metadata"`
}

func (s *ShareUri) FileName() string {
	return filepath.Base(s.Key)
}

func (s *ShareUri) ResourceKey() string {
	k := strings.Split(s.Key, "/")
	p := k[:len(k)-1]
	return strings.Join(p, "/")
}

func (s *ShareUri) String() (string, error) {
	return CreateShareUri(s)
}

func CreateShareUri(s *ShareUri) (string, error) {
	sid := Base64Encode([]byte(s.Session))
	return JoinUrl(fmt.Sprintf(
		"/share/%s?ttl=%d&session=%s",
		strings.Join([]string{s.Bucket, s.Key}, "/"),
		s.TTL,
		sid,
	))
}

func ParseShareUri(u *url.URL) (*ShareUri, error) {
	p := strings.TrimPrefix(u.Path, "/share/")

	s, _ := Base64Decode(u.Query().Get("session"))
	ttl, _ := strconv.ParseInt(u.Query().Get("ttl"), 10, 32)

	pth := strings.Split(p, "/")
	b := pth[0]
	k := filepath.Join(pth[1:]...)

	return &ShareUri{
		Session:  string(s),
		TTL:      time.Duration(ttl),
		Bucket:   b,
		Key:      k,
		Metadata: Metadata{},
	}, nil
}
