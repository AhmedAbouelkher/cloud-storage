package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

func HandleObjectCreation(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(MaxUploadLimit); err != nil {
		SendHttpJsonError(w, http.StatusBadRequest, err)
		return

	}
	f, h, err := r.FormFile("file")
	if err != nil {
		SendHttpJsonError(w, http.StatusBadRequest, err)
		return
	}
	defer f.Close()
	defer r.Body.Close()

	b := r.FormValue("bucket")
	if b == "" {
		SendHttpJsonError(w, http.StatusUnprocessableEntity, errors.New("bucket name is required"))
		return
	}

	fPath := r.FormValue("key")
	if fPath == "" {
		fPath = h.Filename
	}

	cfg := &SaveConfig{
		Bucket:   b,
		Reader:   f,
		FilePath: fPath,
		Mime:     h.Header.Get("Content-Type"),
		Ext:      strings.TrimPrefix(filepath.Ext(fPath), "."),
	}
	o := &Object{}

	if _, err := o.Save(cfg); err != nil {
		SendHttpJsonError(w, http.StatusInternalServerError, err)
		return
	}

	SendJson(w, http.StatusOK, Payload{
		"message": "object created",
		"object":  o,
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

	o, err := FindObject(uuid)
	if err != nil {
		SendHttpJsonError(w, http.StatusInternalServerError, err)
		return
	}

	k, _ := o.Key()
	s3, _ := o.S3()

	SendJson(w, http.StatusOK, Payload{
		"object": o,
		"key":    k,
		"s3_uri": s3,
	})
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
	ttl, _ := strconv.ParseInt(uTTL, 10, 32)

	if uuid == "" {
		SendHttpJsonError(w, http.StatusUnprocessableEntity, errors.New("uuid is required"))
		return
	}

	if ttl > (24 * 60 * 60) {
		SendHttpJsonError(w, http.StatusUnprocessableEntity, errors.New("ttl is too long"))
		return
	}

	o, err := FindObject(uuid)
	if err != nil {
		SendHttpJsonError(w, http.StatusInternalServerError, err)
		return
	}

	l, s, err := o.GenerateSharableLink(&ObjectShare{
		TTL: time.Duration(ttl),
	})
	if err != nil {
		SendHttpJsonError(w, http.StatusInternalServerError, err)
		return
	}

	SendJson(w, http.StatusOK, Payload{
		"url":        l,
		"uuid":       s.EntityTag,
		"ttl":        s.TTL,
		"session_id": s.ID,
		"expire_at":  s.ExpiryDate.Format(time.RFC3339),
	})
}

func HandleServingRequestedObject(w http.ResponseWriter, r *http.Request) {
	// uuid := mux.Vars(r)["uuid"]
	// session := r.URL.Query().Get("session")

	// if uuid == "" || session == "" {
	// 	SendHttpJsonError(w, http.StatusUnprocessableEntity, errors.New("uuid and session are required"))
	// 	return
	// }

	// uuid = NameWithoutExt(uuid) //remove extension from uuid
	// f, err := ServeObject(uuid, session)
	// if err != nil {
	// 	if errors.Is(err, ErrSessionExpired) {
	// 		SendHttpJsonError(w, http.StatusForbidden, err)
	// 		return
	// 	} else if errors.Is(err, ErrSessionNotFound) {
	// 		SendHttpJsonError(w, http.StatusUnauthorized, err)
	// 		return
	// 	}
	// 	SendHttpJsonError(w, http.StatusInternalServerError, err)
	// 	return
	// }

	// defer f.Close()

	// w.Header().Set("Content-Type", f.Type)
	// http.ServeContent(w, r, f.Name(), time.Time{}, f.File)

	SendJson(w, http.StatusOK, Payload{
		"path": r.URL.Path,
		"dir":  r.URL.RawPath,
		"q":    r.URL.Query(),
	})
}

func HandleFileUpload(w http.ResponseWriter, r *http.Request) {
	if IsProduction() {
		SendHttpJsonError(w, http.StatusUnauthorized, errors.New("access is not allowed"))
		return
	}

	if err := r.ParseMultipartForm(MaxUploadLimit); err != nil {
		SendHttpJsonError(w, http.StatusBadRequest, err)
		return
	}

	f, h, err := r.FormFile("file")
	if err != nil {
		SendHttpJsonError(w, http.StatusUnprocessableEntity, err)
		return
	}
	defer f.Close()
	defer r.Body.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, f); err != nil {
		SendHttpJsonError(w, http.StatusInternalServerError, err)
		return
	}

	c := buf.String()
	if h.Size > 1024 {
		c = c[:1024]
	}

	SendJson(w, http.StatusOK, Payload{
		"message": "object uploaded",
		"name": Payload{
			"original":  h.Filename,
			"extension": h.Header.Get("Content-Type"),
			"size":      h.Size,
		},
		"content": c,
	})
}

func HandleObjectsCreation(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(MaxUploadLimit); err != nil {
		SendHttpJsonError(w, http.StatusBadRequest, err)
		return

	}

	form := r.MultipartForm
	files := form.File["files"]
	if len(files) == 0 {
		SendHttpJsonError(w, http.StatusBadRequest, errors.New("no files provided"))
		return
	}

	b := r.FormValue("bucket")
	if b == "" {
		SendHttpJsonError(w, http.StatusUnprocessableEntity, errors.New("bucket name is required"))
		return
	}

	subP := r.FormValue("subpath")
	if subP != "" {
		subP = path.Clean(subP)
		if !path.IsAbs(subP) {
			SendHttpJsonError(w, http.StatusUnprocessableEntity, errors.New("subpath must be absolute"))
			return
		}
	}

	scfg := SaveMultipleConfig{
		BucketID: b,
	}

	for _, fh := range files {
		f, err := fh.Open()
		if err != nil {
			SendHttpJsonError(w, http.StatusInternalServerError, err)
			return
		}
		defer f.Close()

		key := filepath.Join(subP, fh.Filename)

		cfg := &SaveConfig{
			Reader:   f,
			FilePath: key,
		}
		o := &Object{
			Type: &ObjectType{
				Mime: fh.Header.Get("Content-Type"),
			},
		}
		scfg.Push(o, cfg)
	}

	objs, err := scfg.Save()
	if err != nil {
		SendHttpJsonError(w, http.StatusInternalServerError, err)
		return
	}

	s3Uri := BuildS3Path(&S3Path{
		Bucket:  b,
		RawPath: subP,
	})

	resp := UploadedObjectsResponse{
		Bucket:  b,
		Subpath: subP,
		Path:    s3Uri,
	}
	resp.FromObjects(objs)

	SendJson(w, http.StatusOK, resp)
}
