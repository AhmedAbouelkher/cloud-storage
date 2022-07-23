package main

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

func HandleObjectCreation(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(MaxUploadLimit); err != nil {
		SendHttpJsonError(w, http.StatusBadRequest, err)
		return
	}

	b := r.FormValue("bucket")
	if b == "" {
		SendHttpJsonError(w, http.StatusUnprocessableEntity, errors.New("bucket name is required"))
		return
	}

	f, h, err := r.FormFile("file")
	if err != nil {
		SendHttpJsonError(w, http.StatusBadRequest, err)
		return
	}
	defer f.Close()

	k := r.FormValue("key")
	if k == "" {
		k = h.Filename
	}

	cfg := &SaveConfig{
		BucketID: r.FormValue("bucket"),
		Reader:   f,
		Key:      k,
	}
	o := &Object{
		Type: h.Header.Get("Content-Type"),
	}

	if _, err := o.Save(cfg); err != nil {
		SendHttpJsonError(w, http.StatusInternalServerError, err)
		return
	}

	SendJson(w, http.StatusOK, Payload{
		"message": "object created",
		"uuid":    o.UUID,
	})
}

func HandleObjectFetch(w http.ResponseWriter, r *http.Request) {
	if IsProduction() {
		SendHttpJsonError(w, http.StatusUnauthorized, errors.New("access is not allowed"))
		return
	}

	uuid := mux.Vars(r)["uuid"]
	if uuid == "" {
		SendHttpJsonError(w, http.StatusUnprocessableEntity, errors.New("uuid is required"))
		return
	}

	o, err := FetchObject(uuid)
	if err != nil {
		SendHttpJsonError(w, http.StatusInternalServerError, err)
		return
	}

	SendJson(w, http.StatusOK, Payload{"object": o})
}

func HandleObjectDeletion(w http.ResponseWriter, r *http.Request) {
	uuid := mux.Vars(r)["uuid"]
	if uuid == "" {
		SendHttpJsonError(w, http.StatusUnprocessableEntity, errors.New("uuid is required"))
		return
	}

	if err := DeleteObject(uuid); err != nil {
		SendHttpJsonError(w, http.StatusInternalServerError, err)
		return
	}

	SendJson(w, http.StatusOK, Payload{"message": "object deleted"})
}

func HandleGeneratingSharableLink(w http.ResponseWriter, r *http.Request) {
	uuid := mux.Vars(r)["uuid"]
	// extract ttl from query param
	uTTL := r.URL.Query().Get("ttl")

	if uuid == "" {
		SendHttpJsonError(w, http.StatusUnprocessableEntity, errors.New("uuid is required"))
		return
	}

	o, err := FetchObject(uuid)
	if err != nil {
		SendHttpJsonError(w, http.StatusInternalServerError, err)
		return
	}

	ttl, _ := strconv.ParseInt(uTTL, 10, 64)
	l, s, err := o.GenerateSharableLink(&ObjectShare{
		TTL: time.Duration(ttl),
	})
	if err != nil {
		SendHttpJsonError(w, http.StatusInternalServerError, err)
		return
	}

	SendJson(w, http.StatusOK, Payload{
		"url":        l,
		"uuid":       s.OUUID,
		"ttl":        s.TTL,
		"session_id": s.ID,
		"expire_at":  s.ExpiryDate.Format(time.RFC3339),
	})
}

func HandleServingRequestedObject(w http.ResponseWriter, r *http.Request) {
	uuid := mux.Vars(r)["uuid"]
	session := r.URL.Query().Get("session")

	if uuid == "" || session == "" {
		SendHttpJsonError(w, http.StatusUnprocessableEntity, errors.New("uuid and session are required"))
		return
	}

	uuid = NameWithoutExt(uuid) //remove extension from uuid
	f, err := ServeObject(uuid, session)
	if err != nil {
		if errors.Is(err, ErrSessionExpired) {
			SendHttpJsonError(w, http.StatusForbidden, err)
			return
		} else if errors.Is(err, ErrSessionNotFound) {
			SendHttpJsonError(w, http.StatusUnauthorized, err)
			return
		}
		SendHttpJsonError(w, http.StatusInternalServerError, err)
		return
	}

	defer f.Close()

	w.Header().Set("Content-Type", f.Type)

	http.ServeContent(w, r, f.Name(), time.Time{}, f.File)
}
