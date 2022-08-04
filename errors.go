package main

import "errors"

var (
	ErrSessionExpired   = errors.New("session expired")
	ErrSessionNotFound  = errors.New("session not found")
	ErrInvalidSession   = errors.New("invalid session")
	ErrBucketNotFound   = errors.New("bucket not found")
	ErrResourceNotFound = errors.New("resource not found")
	ErrObjectNotFound   = errors.New("object not found")
)
