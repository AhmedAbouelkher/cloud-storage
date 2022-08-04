package main

import (
	"net/http"
)

type resourcePayload struct {
	Name   string `json:"name" validate:"required,min=5,max=256"`
	Bucket string `json:"bucket" validate:"required"`
}

func HandleResourceCreation(w http.ResponseWriter, r *http.Request) {
	var payload resourcePayload
	if err := ParseAndValidate(r, &payload); err != nil {
		SendValidationError(w, err, http.StatusUnprocessableEntity)
		return
	}
	defer r.Body.Close()

	rsrc := &Resource{
		Name: payload.Name,
	}

	if err := rsrc.Create(payload.Bucket); err != nil {
		SendHttpJsonError(w, http.StatusBadRequest, err)
		return
	}

	s3, _ := rsrc.S3Path()

	SendJson(w, http.StatusCreated, Payload{
		"message":  "resource created",
		"resource": rsrc,
		"s3_path":  s3,
	})
}

type resourceS3FetchPayload struct {
	S3Path string `json:"s3_path" validate:"required"`
}

func HandleResourceFetchWithS3(w http.ResponseWriter, r *http.Request) {
	var payload resourceS3FetchPayload
	if err := ParseAndValidate(r, &payload); err != nil {
		SendValidationError(w, err, http.StatusUnprocessableEntity)
		return
	}
	defer r.Body.Close()

	s3, err := ParseS3Path(payload.S3Path)
	if err != nil {
		SendHttpJsonError(w, http.StatusBadRequest, err)
		return
	}

	rsrc := &Resource{}

	if err := FindWithS3(s3, rsrc); err != nil {
		SendHttpJsonError(w, http.StatusBadRequest, err)
		return
	}

	SendJson(w, http.StatusOK, Payload{
		"message":  "fetched s3 resource",
		"resource": rsrc,
	})
}

func HandleResourceFilesFetchWithS3(w http.ResponseWriter, r *http.Request) {
	var payload resourceS3FetchPayload
	if err := ParseAndValidate(r, &payload); err != nil {
		SendValidationError(w, err, http.StatusUnprocessableEntity)
		return
	}
	defer r.Body.Close()

	s3, err := ParseS3Path(payload.S3Path)
	if err != nil {
		SendHttpJsonError(w, http.StatusBadRequest, err)
		return
	}

	objs, err := ListObjectsS3(s3)
	if err != nil {
		SendHttpJsonError(w, http.StatusBadRequest, err)
		return
	}

	SendJson(w, http.StatusOK, Payload{
		"message": "fetched s3 resource",
		"objects": objs,
	})
}

type resourceDeletionPayload struct {
	UUID  string `json:"uuid" validate:"required,uuid4"`
	Force bool   `json:"force"`
}

func HandleResourceDeletion(w http.ResponseWriter, r *http.Request) {
	var payload resourceDeletionPayload
	if err := ParseAndValidate(r, &payload); err != nil {
		SendValidationError(w, err, http.StatusUnprocessableEntity)
		return
	}
	defer r.Body.Close()

	rsrc := &Resource{
		UUID: payload.UUID,
	}

	if err := rsrc.Delete(payload.Force); err != nil {
		SendHttpJsonError(w, http.StatusBadRequest, err)
		return
	}

	s3, _ := rsrc.S3Path()

	SendJson(w, http.StatusOK, Payload{
		"message": "resource deleted",
		"s3_path": s3,
	})
}
