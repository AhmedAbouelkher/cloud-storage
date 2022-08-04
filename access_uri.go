package main

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

type AccessUri struct {
	Bucket   string
	Key      string
	Metadata Metadata
}

func (s *AccessUri) FileName() string {
	return filepath.Base(s.Key)
}

func (s *AccessUri) FileSuffix() string {
	return strings.TrimPrefix(filepath.Ext(s.Key), ".")
}

func (s *AccessUri) ResourceKey() string {
	k := strings.Split(s.Key, "/")
	p := k[:len(k)-1]
	return strings.Join(p, "/")
}

func (s *AccessUri) String() (string, error) {
	return createAccessUri(s)
}

func createAccessUri(s *AccessUri) (string, error) {
	return JoinUrl(fmt.Sprintf(
		"/share/%s",
		strings.Join([]string{s.Bucket, s.Key}, "/"),
	))
}

func ParseAccessUri(u *url.URL) (*AccessUri, error) {
	p := strings.TrimPrefix(u.Path, "/")
	pth := strings.Split(p, "/")

	b := pth[0]                     // bucket name
	k := strings.Join(pth[1:], "/") // file full key

	return &AccessUri{
		Bucket:   b,
		Key:      k,
		Metadata: Metadata{},
	}, nil
}
