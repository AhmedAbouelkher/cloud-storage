package main

import (
	"errors"
	"net/http"

	"github.com/gorilla/mux"
)

type bucketPayload struct {
	Name string `json:"name" validate:"required,min=5,max=256"`
}

func HandleBucketCreation(w http.ResponseWriter, r *http.Request) {
	var payload bucketPayload
	if err := ParseAndValidate(r, &payload); err != nil {
		SendValidationError(w, err, http.StatusUnprocessableEntity)
		return
	}
	defer r.Body.Close()

	b := &Bucket{
		Name: payload.Name,
	}

	if err := b.Create(); err != nil {
		SendHttpJsonError(w, http.StatusBadRequest, err)
		return
	}

	SendJson(w, http.StatusCreated, Payload{
		"message": "bucket created",
		"bucket":  b.Name,
	})
}

func HandleBucketDeletion(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if name == "" {
		SendHttpJsonError(w, http.StatusUnprocessableEntity, errors.New("bucket name is required"))
		return
	}

	b := &Bucket{
		Name: name,
	}

	if err := b.Delete(); err != nil {
		SendHttpJsonError(w, http.StatusBadRequest, err)
		return
	}

	SendJson(w, http.StatusOK, Payload{
		"message": "bucket deleted",
	})
}

func HandleObjectsFetch(w http.ResponseWriter, r *http.Request) {
	if IsProduction() {
		SendHttpJsonError(w, http.StatusUnauthorized, errors.New("access is not allowed"))
		return
	}
	name := mux.Vars(r)["name"]
	if name == "" {
		SendHttpJsonError(w, http.StatusUnprocessableEntity, errors.New("bucket name is required"))
		return
	}

	b := &Bucket{
		Name: name,
	}

	obs, err := b.FetchObjects()

	if err != nil {
		SendHttpJsonError(w, http.StatusBadRequest, err)
		return
	}

	SendJson(w, http.StatusOK, Payload{
		"objects": obs,
	})
}
